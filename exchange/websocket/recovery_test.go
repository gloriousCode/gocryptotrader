package websocket

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestConnectionFault(t *testing.T) {
	t.Parallel()
	fault := &connectionFaultError{connection: &connection{URL: "wss://example.test/socket?token=secret"}, cause: errDastardlyReason}
	assert.ErrorIs(t, fault, errConnectionFault, "fault should preserve the connection sentinel")
	assert.ErrorIs(t, fault, errDastardlyReason, "fault should preserve the original failure")
	assert.NotContains(t, fault.Error(), "secret", "fault diagnostics should omit URL credentials")
	assert.Contains(t, fault.Error(), "example.test/socket", "fault diagnostics should identify the endpoint")
}

func TestRecoverConnection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		subscriberFailures   int32
		callbackFailures     int32
		dialFailure          bool
		expireAttempt        bool
		stalledWrite         bool
		initialPartial       bool
		temporaryUnsubscribe bool
		explicitUnsubscribe  bool
		nativeUnsubscribe    bool
	}{
		{name: "successful replacement"},
		{name: "partial subscription failure", subscriberFailures: 1},
		{name: "invalidation retry", callbackFailures: 1},
		{name: "cancel pending dial", dialFailure: true},
		{name: "expired subscription cannot install", expireAttempt: true},
		{name: "cancel stalled subscription write", stalledWrite: true},
		{name: "initial partial failure retains every requested subscription", initialPartial: true},
		{name: "fault after temporary unsubscribe acknowledgement", temporaryUnsubscribe: true},
		{name: "explicit unsubscribe withdraws recovery intent", explicitUnsubscribe: true},
		{name: "explicit native unsubscribe withdraws recovery intent", nativeUnsubscribe: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			url, dialer, open := newConnectionCountingServer(t)
			manager := NewManager()
			setup := newDefaultSetup()
			setup.UseMultiConnectionManagement = true
			require.NoError(t, manager.Setup(setup), "manager setup must succeed")
			manager.setState(connectedState)
			var attempts, invalidations atomic.Int32
			pending := make(chan struct{}, 1)
			desiredCount := 2
			if tc.explicitUnsubscribe || tc.nativeUnsubscribe {
				desiredCount = 1
			}
			var old Connection
			var stalledSocket *connection
			var stalled *stalledWriteConn
			if tc.stalledWrite {
				stalledSocket, stalled = dialStalledWriteConn(t, false)
				stalled.stalled.Store(true)
			}
			connSetup := &ConnectionSetup{
				URL:                  url,
				RecoverIndependently: true,
				MessageFilter:        asset.Spot,
				Connector: func(ctx context.Context, conn Connection) error {
					attempt := attempts.Add(1)
					if attempt > 1 && tc.stalledWrite {
						concrete, ok := conn.(*connection)
						if !assert.True(t, ok, "recovery socket should be a managed connection") {
							return errDastardlyReason
						}
						concrete.Connection = stalledSocket.Connection
						concrete.setConnectedStatus(true)
						return nil
					}
					if attempt > 1 && tc.dialFailure {
						pending <- struct{}{}
						<-ctx.Done()
						return ctx.Err()
					}
					return conn.Dial(ctx, dialer, http.Header{}, nil)
				},
				GenerateSubscriptions: func() (subscription.List, error) { return nil, nil },
				Subscriber: func(ctx context.Context, conn Connection, subs subscription.List) error {
					if attempts.Load() == 1 && tc.initialPartial {
						if err := manager.AddSuccessfulSubscriptions(conn, subs[0]); err != nil {
							return err
						}
						return errDastardlyReason
					}

					if attempts.Load() > 1 && tc.stalledWrite {
						return conn.SendRawMessage(ctx, 0, 1, []byte("stalled subscription"))
					}
					if attempts.Load() > 1 && tc.expireAttempt {
						<-ctx.Done()
						return nil
					}
					if attempts.Load() > 1 && attempts.Load() <= 1+tc.subscriberFailures {
						if err := manager.AddSuccessfulSubscriptions(conn, subs[0]); err != nil {
							return err
						}
						return errDastardlyReason
					}
					return manager.AddSuccessfulSubscriptions(conn, subs...)
				},
				Unsubscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
					return manager.RemoveSubscriptions(conn, subs...)
				},
				Handler: func(context.Context, Connection, []byte) error { return nil },
				OnConnectionFault: func(conn Connection, subs subscription.List) error {
					assert.NotNil(t, conn, "invalidation should identify the retired socket")
					if invalidations.Load() == 0 {
						assert.Same(t, old, conn, "first invalidation should identify the original failed socket")
					}
					assert.Len(t, subs, desiredCount, "invalidation should receive the complete desired subscriptions")
					if invalidations.Add(1) <= tc.callbackFailures {
						return errDastardlyReason
					}
					return nil
				},
			}
			manager.features.Unsubscribe = false
			require.NoError(t, manager.SetupNewConnection(connSetup), "connection setup must succeed")
			ws := manager.snapshotConnectionManager()[0]
			subs := subscription.List{
				{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
			}
			initialErr := manager.createConnectAndSubscribe(t.Context(), ws, subs)
			if tc.initialPartial {
				require.ErrorIs(t, initialErr, errDastardlyReason, "initial partial subscription must preserve its failure")
			} else {
				require.NoError(t, initialErr, "initial connection must succeed")
			}
			old = manager.snapshotManagedConnections(ws)[0]
			t.Cleanup(func() { require.NoError(t, manager.Shutdown(), "manager shutdown must drain all recovery workers") })
			concrete, ok := old.(*connection)
			require.True(t, ok, "initial socket must be a managed connection")
			if tc.temporaryUnsubscribe {
				require.NoError(t, subs[1].SetState(subscription.ResubscribingState), "temporary unsubscribe must mark that the feed is still requested")
				require.NoError(t, manager.RemoveSubscriptions(old, subs[1]), "temporary Gate unsubscribe acknowledgement must update accepted state")
				assert.Len(t, concrete.desiredSubscriptions.List(), 2, "temporary unsubscribe acknowledgement should retain the requested market")
			}
			if tc.nativeUnsubscribe {
				require.NoError(t, manager.RemoveSubscriptions(old, subs[1]), "native explicit unsubscribe acknowledgement must withdraw its feed")
				assert.Len(t, concrete.desiredSubscriptions.List(), 1, "native explicit removal should withdraw the requested market")
			}
			if tc.explicitUnsubscribe {
				require.NoError(t, manager.UnsubscribeChannels(t.Context(), old, subs[1:]), "explicit unsubscribe intent must succeed")
				assert.Len(t, concrete.desiredSubscriptions.List(), 1, "explicit removal should withdraw the requested market")
			}

			if tc.initialPartial {
				assert.Len(t, old.Subscriptions().List(), 1, "initial store should contain only acknowledged subscriptions")
				assert.Len(t, concrete.desiredSubscriptions.List(), 2, "recovery ownership should retain all requested subscriptions")
				select {
				case fault := <-manager.ReadMessageErrors:
					assert.ErrorIs(t, fault, errConnectionFault, "partial subscription should trigger connection recovery without waiting for a reader failure")
				default:
					require.FailNow(t, "initial partial failure must publish its recovery notification")
				}
			} else {
				require.NoError(t, concrete.Connection.Close(), "closing the peer socket must succeed")
			}
			require.Eventually(t, concrete.faulted.Load, 5*time.Second, time.Millisecond, "reader must retain the failure")
			recoveryCtx := t.Context()
			if tc.expireAttempt || tc.stalledWrite {
				var cancel context.CancelFunc
				recoveryCtx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
				defer cancel()
			}
			require.True(t, manager.recoverConnectionFaults(recoveryCtx, &connectionFaultError{connection: concrete, cause: errDastardlyReason}), "socket failure must use independent recovery")
			if tc.expireAttempt || tc.stalledWrite {
				manager.recoveryWg.Wait()
				if tc.stalledWrite {
					select {
					case <-stalled.blocked:
					default:
						assert.Fail(t, "subscriber should reach the stalled write before cancellation")
					}
				}
				conns := manager.snapshotManagedConnections(ws)
				require.Len(t, conns, 1, "expired attempt must leave only the original recovery placeholder")
				assert.Same(t, old, conns[0], "expired subscription attempt should never be installed")
				assert.False(t, conns[0].IsConnected(), "expired attempt should fail closed")
				return
			}
			if tc.dialFailure {
				select {
				case <-pending:
				case <-time.After(5 * time.Second):
					t.Fatal("recovery dial must start")
				}
				// Acquiring this method's lifecycle lock proves the pending network operation does not hold it.
				_, err := manager.GetConnection(asset.Spot)
				require.ErrorIs(t, err, ErrNotConnected, "pending replacement must fail closed")
				require.ErrorIs(t, manager.scaleConnectionsToSubscriptions(t.Context(), ws, nil), ErrConnectionRecovering, "subscription mutation must fail while recovery owns desired state")
				require.ErrorIs(t, manager.UnsubscribeChannels(t.Context(), old, subs), ErrConnectionRecovering, "unsubscription must preserve desired state during recovery")
				require.ErrorIs(t, manager.ResubscribeFromConnection(t.Context(), old, subs), ErrConnectionRecovering, "resubscription must preserve desired state during recovery")

				return
			}
			require.Eventually(t, func() bool {
				conns := manager.snapshotManagedConnections(ws)
				return len(conns) == 1 && conns[0] != old && conns[0].IsConnected()
			}, 8*time.Second, time.Millisecond, "recovery must install a live replacement")
			replacement := manager.snapshotManagedConnections(ws)[0]
			assert.Len(t, replacement.Subscriptions().List(), desiredCount, "replacement should restore all desired subscriptions after partial failure")
			assert.Len(t, ws.subscriptions.List(), desiredCount, "manager store should hold only the replacement subscriptions")
			assert.Equal(t, 1+tc.callbackFailures+tc.subscriberFailures, invalidations.Load(), "successful invalidation should run once per retired socket, including failed partial attempts")
			if tc.callbackFailures > 0 || tc.subscriberFailures > 0 {
				select {
				case result := <-manager.DataHandler.C:
					reported, ok := result.Data.(error)
					require.True(t, ok, "recovery diagnostics must retain an error value")
					assert.ErrorIs(t, reported, errDastardlyReason, "recovery diagnostics should preserve wrapped failure identity")
				case <-time.After(time.Second):
					require.FailNow(t, "recovery failure must reach DataHandler")
				}
			}
			require.NoError(t, replacement.SendRawMessage(t.Context(), 0, 1, []byte("still alive")), "replacement must retain a live reader context")
			assert.Eventually(t, func() bool { return open.Load() == 1 }, 5*time.Second, time.Millisecond, "failed attempts should release every retired socket")
		})
	}
}

func TestRecoverConnectionFaults(t *testing.T) {
	t.Parallel()
	t.Run("coalesced required fault keeps full shutdown semantics", func(t *testing.T) {
		t.Parallel()
		manager := NewManager()
		setup := newDefaultSetup()
		setup.UseMultiConnectionManagement = true
		require.NoError(t, manager.Setup(setup), "manager setup must succeed")
		manager.setState(connectedState)
		scoped := manager.createConnectionFromSetup(&ConnectionSetup{})
		required := manager.createConnectionFromSetup(&ConnectionSetup{})
		scoped.faulted.Store(true)
		required.faulted.Store(true)
		optionalWS := &websocket{setup: &ConnectionSetup{RecoverIndependently: true}, subscriptions: subscription.NewStore()}
		requiredWS := &websocket{setup: &ConnectionSetup{}, subscriptions: subscription.NewStore()}
		manager.connectionManager = []*websocket{optionalWS, requiredWS}
		manager.trackConnection(scoped, optionalWS)
		manager.trackConnection(required, requiredWS)
		assert.False(t, manager.recoverConnectionFaults(t.Context(), &connectionFaultError{connection: scoped, cause: errDastardlyReason}), "required socket failure should force the existing full recovery path")
		assert.False(t, scoped.recovering, "optional recovery should not race a required full shutdown")
	})
	t.Run("legacy connections retain full reconnect", func(t *testing.T) {
		t.Parallel()
		manager := NewManager()
		require.NoError(t, manager.Setup(newDefaultSetup()), "legacy manager setup must succeed")
		manager.setState(connectedState)
		manager.connectionMonitorRunning.Store(true)
		var attempts atomic.Int32
		manager.connector = func() error { attempts.Add(1); return errDastardlyReason }
		manager.Conn = manager.createConnectionFromSetup(&ConnectionSetup{})
		concrete, ok := manager.Conn.(*connection)
		require.True(t, ok, "legacy socket must be a managed connection")
		manager.ReadMessageErrors <- &connectionFaultError{connection: concrete, cause: errDastardlyReason}
		timer := time.NewTimer(time.Hour)
		defer timer.Stop()
		require.False(t, manager.observeConnection(t.Context(), timer), "observer must continue after reconnect attempt")
		assert.EqualValues(t, 1, attempts.Load(), "legacy fault should invoke the full connector")
		assert.False(t, manager.IsConnected(), "failed legacy reconnect should leave the manager disconnected")
	})
	url, dialer, _ := newConnectionCountingServer(t)
	manager := NewManager()
	setup := newDefaultSetup()
	setup.UseMultiConnectionManagement = true
	require.NoError(t, manager.Setup(setup), "setup must succeed")
	manager.features.Unsubscribe = false
	manager.setState(connectedState)
	for _, filter := range []asset.Item{asset.Spot, asset.Futures, asset.Options} {
		require.NoError(t, manager.SetupNewConnection(&ConnectionSetup{
			URL: url, MessageFilter: filter, RecoverIndependently: true, SubscriptionsNotRequired: true,
			Connector: func(ctx context.Context, conn Connection) error { return conn.Dial(ctx, dialer, nil, nil) },
			Handler:   func(context.Context, Connection, []byte) error { return nil },
		}), "connection setup must succeed")
	}
	original := make([]Connection, 3)
	for i, ws := range manager.snapshotConnectionManager() {
		require.NoError(t, manager.createConnectAndSubscribe(t.Context(), ws, nil), "initial connection must succeed")
		original[i] = manager.snapshotManagedConnections(ws)[0]
	}
	t.Cleanup(func() { require.NoError(t, manager.Shutdown(), "shutdown must succeed") })
	// Occupy the one-slot error channel so neither failure can depend on delivery.
	manager.ReadMessageErrors <- errDastardlyReason
	for _, conn := range original[:2] {
		concrete, ok := conn.(*connection)
		require.True(t, ok, "initial socket must be a managed connection")
		require.NoError(t, concrete.Connection.Close(), "closing failed socket must succeed")
		require.Eventually(t, concrete.faulted.Load, 5*time.Second, time.Millisecond, "reader must retain dropped fault notification")
	}
	manager.recoverConnectionFaults(t.Context(), nil)
	require.Eventually(t, func() bool {
		for i, ws := range manager.snapshotConnectionManager()[:2] {
			conns := manager.snapshotManagedConnections(ws)
			if len(conns) != 1 || conns[0] == original[i] || !conns[0].IsConnected() {
				return false
			}
		}
		return true
	}, 5*time.Second, time.Millisecond, "periodic scan must recover both coalesced failures")
	healthy, err := manager.GetConnection(asset.Options)
	require.NoError(t, err, "healthy feed must remain available")
	assert.Same(t, original[2], healthy, "unaffected socket should retain its identity")
	assert.True(t, manager.IsConnected(), "manager should remain connected throughout scoped recovery")
	// A delayed error belonging to a retired generation must not reset the healthy manager.
	retired, ok := original[0].(*connection)
	require.True(t, ok, "retired socket must be a managed connection")
	assert.True(t, manager.recoverConnectionFaults(t.Context(), &connectionFaultError{connection: retired, cause: errDastardlyReason}), "retired fault should be recognised as scoped")
}

func TestDiscardRecoveryAttempt(t *testing.T) {
	t.Parallel()
	manager, conn := newManagedSubscriptionTestManager(t)
	ws, ok := manager.managedWebsocket(conn)
	require.True(t, ok, "test connection must be registered")
	manager.discardRecoveryAttempt(ws)
	_, ok = manager.managedWebsocket(conn)
	assert.False(t, ok, "discard should remove pending attempt routing")
	assert.Empty(t, conn.Subscriptions().List(), "discard should clear connection subscriptions")
	assert.Empty(t, ws.subscriptions.List(), "discard should clear pending subscription state")
}

func TestConnectionRecoveryTimeout(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		count            int
		responseMaxLimit time.Duration
		expected         time.Duration
	}{
		{name: "no subscriptions", expected: defaultHandshakeTimeout},
		{name: "negative count", count: -1, expected: defaultHandshakeTimeout},
		{name: "single acknowledgement", count: 1, responseMaxLimit: 7 * time.Second, expected: 52 * time.Second},
		{name: "full market set", count: 1500, responseMaxLimit: 7 * time.Second, expected: defaultHandshakeTimeout + 10500*time.Second},
		{name: "unspecified acknowledgement limit", count: 2, expected: 3 * defaultHandshakeTimeout},
		{name: "overflow saturates", count: 2, responseMaxLimit: time.Duration(1<<63 - 1), expected: time.Duration(1<<63 - 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, connectionRecoveryTimeout(tc.count, tc.responseMaxLimit), "recovery budget should allow all desired acknowledgements without overflow")
		})
	}
}

func TestReportFault(t *testing.T) {
	t.Parallel()
	for _, full := range []bool{false, true} {
		t.Run(map[bool]string{false: "delivered", true: "coalesced"}[full], func(t *testing.T) {
			t.Parallel()
			conn := &connection{readMessageErrors: make(chan error, 1)}
			if full {
				conn.readMessageErrors <- errDastardlyReason
			}
			conn.reportFault(errDastardlyReason)
			assert.True(t, conn.faulted.Load(), "fault state should survive a full notification channel")
			if !full {
				reported := <-conn.readMessageErrors
				assert.ErrorIs(t, reported, errConnectionFault, "notification should retain the lifecycle sentinel")
				assert.ErrorIs(t, reported, errDastardlyReason, "notification should retain the original failure")
			}
		})
	}
}
