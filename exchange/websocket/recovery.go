package websocket

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// ErrConnectionRecovering prevents subscription changes from racing replacement of a failed socket.
var ErrConnectionRecovering = errors.New("websocket connection recovery is in progress")

// connectionFaultError retains socket identity while preserving errors.Is compatibility.
type connectionFaultError struct {
	connection *connection
	cause      error
}

func (f *connectionFaultError) Error() string {
	return fmt.Sprintf("%s: %s (%q)", f.cause, errConnectionFault, removeURLQueryString(f.connection.URL))
}

func (f *connectionFaultError) Unwrap() []error { return []error{f.cause, errConnectionFault} }

// recoverConnectionFaults scans retained fault flags because the shared notification channel can coalesce failures.
// The lifecycle lock is held only while registering workers; network operations run independently.
func (m *Manager) recoverConnectionFaults(ctx context.Context, reported error) bool {
	if !m.useMultiConnectionManagement {
		return false
	}
	m.m.Lock()
	defer m.m.Unlock()
	var fault *connectionFaultError
	scoped := errors.As(reported, &fault)
	if scoped {
		ws, tracked := m.managedWebsocket(fault.connection)
		if tracked && !ws.setup.RecoverIndependently {
			return false
		}
	}
	if !m.IsConnected() || !m.IsEnabled() {
		return scoped
	}
	for _, ws := range m.snapshotConnectionManager() {
		if ws.setup.RecoverIndependently {
			continue
		}
		for _, candidate := range m.snapshotManagedConnections(ws) {
			if conn, ok := candidate.(*connection); ok && conn.faulted.Load() {
				if reported == nil {
					go func() {
						if err := m.Shutdown(); err != nil {
							log.Errorln(log.WebsocketMgr, err)
						}
					}()
				}
				return false
			}
		}
	}
	for _, ws := range m.snapshotConnectionManager() {
		if !ws.setup.RecoverIndependently {
			continue
		}
		for _, candidate := range m.snapshotManagedConnections(ws) {
			conn, ok := candidate.(*connection)
			if !ok || !conn.faulted.Load() || conn.recovering {
				continue
			}
			conn.recovering = true
			shutdown := m.ShutdownC
			m.recoveryWg.Go(func() { m.recoverConnection(ctx, shutdown, ws, conn) })
		}
	}
	return scoped
}

// recoverConnection owns attempts until successful installation or cancellation of the manager generation.
// Keeping attempts in a separate websocket lets subscription acknowledgements route to an isolated store.
func (m *Manager) recoverConnection(ctx context.Context, shutdown <-chan struct{}, ws *websocket, old *connection) {
	ctx, cancel := context.WithCancel(ctx)
	installed := false
	defer func() {
		if !installed {
			cancel()
		}
	}()
	go func() {
		select {
		case <-shutdown:
			cancel()
		case <-ctx.Done():
		}
	}()
	_ = old.Shutdown()
	old.workers.Wait()
	subs := old.Subscriptions().List()
	if old.desiredSubscriptions != nil {
		subs = old.desiredSubscriptions.List()
	}
	faultedConnection := Connection(old)
	faultedSubscriptions := subs
	invalidated := ws.setup.OnConnectionFault == nil
	delay := time.Second
	for ctx.Err() == nil {
		attempt := &websocket{setup: ws.setup, subscriptions: subscription.NewStore(), recoveryContext: ctx, attemptComplete: make(chan struct{})}
		desired := subs.Clone()
		for _, sub := range desired {
			_ = sub.SetState(subscription.InactiveState)
		}
		var err error
		if !invalidated {
			err = ws.setup.OnConnectionFault(faultedConnection, faultedSubscriptions)
			invalidated = err == nil
		}
		attemptCtx, attemptCancel := context.WithTimeout(ctx, connectionRecoveryTimeout(len(desired), ws.setup.ResponseMaxLimit))
		if err == nil {
			err = m.createConnectAndSubscribe(attemptCtx, attempt, desired)
		}
		attempt.attemptMu.Lock()
		if err == nil {
			err = attemptCtx.Err()
		}
		if err == nil {
			for _, replacement := range m.snapshotManagedConnections(attempt) {
				if !replacement.IsConnected() {
					err = ErrNotConnected
					break
				}
			}
		}
		if err == nil {
			close(attempt.attemptComplete)
		}
		attempt.attemptMu.Unlock()
		attemptCancel()
		if err == nil {
			m.connectionManagerMu.Lock()
			select {
			case <-shutdown:
				err = context.Canceled
			default:
				if m.connections[old] != ws {
					err = ErrNotConnected
				}
				for _, replacement := range attempt.connections {
					if !replacement.IsConnected() {
						err = ErrNotConnected
						break
					}
				}
			}
			if err == nil {
				for _, sub := range subs {
					_ = ws.subscriptions.Remove(sub)
				}
				for _, sub := range attempt.subscriptions.List() {
					if addErr := ws.subscriptions.Add(sub); addErr != nil {
						err = common.AppendError(err, addErr)
					}
				}
				if err == nil {
					delete(m.connections, old)
					ws.connections = slices.DeleteFunc(ws.connections, func(conn Connection) bool { return conn == old })
					for _, replacement := range attempt.connections {
						m.connections[replacement] = ws
						ws.connections = append(ws.connections, replacement)
					}
				}
			}
			m.connectionManagerMu.Unlock()
			if err == nil {
				old.Subscriptions().Clear()
				installed = true
				return
			}
		}
		failedConnections := m.snapshotManagedConnections(attempt)
		m.discardRecoveryAttempt(attempt)
		if len(failedConnections) != 0 && ws.setup.OnConnectionFault != nil {
			// An acknowledged subset may already have published data before another subscription failed.
			faultedConnection = failedConnections[0]
			faultedSubscriptions = desired
			invalidationErr := ws.setup.OnConnectionFault(faultedConnection, faultedSubscriptions)
			invalidated = invalidationErr == nil
			err = common.AppendError(err, invalidationErr)
		}
		if ctx.Err() != nil {
			return
		}
		log.Errorf(log.WebsocketMgr, "%s websocket connection recovery: %s", m.exchangeName, err)
		reportCtx, reportCancel := context.WithTimeout(ctx, time.Second)
		if reportErr := m.DataHandler.Send(reportCtx, err); reportErr != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket connection recovery reporting: %s", m.exchangeName, reportErr)
		}
		reportCancel()
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		delay = min(delay*2, 30*time.Second)
	}
}

// discardRecoveryAttempt releases partial authentication and subscriptions before another attempt can start.
func (m *Manager) discardRecoveryAttempt(attempt *websocket) {
	for _, conn := range m.snapshotManagedConnections(attempt) {
		_ = conn.Shutdown()
		if concrete, ok := conn.(*connection); ok {
			concrete.workers.Wait()
		}
		m.connectionManagerMu.Lock()
		delete(m.connections, conn)
		m.connectionManagerMu.Unlock()
		conn.Subscriptions().Clear()
	}
	attempt.subscriptions.Clear()
}

// connectionRecoveryTimeout allows sequential subscription acknowledgements without assuming an exchange's concurrency.
func connectionRecoveryTimeout(count int, responseMaxLimit time.Duration) time.Duration {
	if count <= 0 {
		return defaultHandshakeTimeout
	}
	if responseMaxLimit <= 0 {
		responseMaxLimit = defaultHandshakeTimeout
	}
	const maximum = time.Duration(1<<63 - 1)
	if int64(count) > int64((maximum-defaultHandshakeTimeout)/responseMaxLimit) {
		return maximum
	}
	return defaultHandshakeTimeout + time.Duration(count)*responseMaxLimit
}
