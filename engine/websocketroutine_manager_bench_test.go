package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var (
	errBenchNilPayload = errors.New("nil payload")
	benchErrSink       uint64
)

func benchHandler(_ string, payload any) error {
	if payload == nil {
		return errBenchNilPayload
	}
	return nil
}

func newBenchmarkManager(tb testing.TB, handlerCount int) *WebsocketRoutineManager {
	tb.Helper()
	m := &WebsocketRoutineManager{}
	if handlerCount <= 0 {
		return m
	}
	if err := m.setWebsocketDataHandler(benchHandler); err != nil {
		tb.Fatalf("setWebsocketDataHandler: %v", err)
	}
	for i := 1; i < handlerCount; i++ {
		if err := m.registerWebsocketDataHandler(benchHandler, false); err != nil {
			tb.Fatalf("registerWebsocketDataHandler: %v", err)
		}
	}
	return m
}

func dispatchHandlers(m *WebsocketRoutineManager, payload any) {
	switch v := any(&m.dataHandlers).(type) {
	case *[]WebsocketDataHandler:
		m.mu.RLock()
		handlers := *v
		for i := range handlers {
			if err := handlers[i]("bench", payload); err != nil {
				atomic.AddUint64(&benchErrSink, 1)
			}
		}
		m.mu.RUnlock()
	case *atomic.Value:
		handlersValue := v.Load()
		if handlersValue == nil {
			return
		}
		handlers := handlersValue.([]WebsocketDataHandler)
		for i := range handlers {
			if err := handlers[i]("bench", payload); err != nil {
				atomic.AddUint64(&benchErrSink, 1)
			}
		}
	default:
		panic("unsupported dataHandlers type")
	}
}

func BenchmarkWebsocketDataHandlerDispatch(b *testing.B) {
	payload := &ticker.Price{ExchangeName: "bench"}
	handlerCounts := []int{1, 4, 16, 64, 256}
	for _, count := range handlerCounts {
		b.Run(fmt.Sprintf("handlers=%d", count), func(b *testing.B) {
			m := newBenchmarkManager(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dispatchHandlers(m, payload)
			}
		})
	}
}

func BenchmarkWebsocketDataHandlerDispatchParallel(b *testing.B) {
	payload := &ticker.Price{ExchangeName: "bench"}
	handlerCounts := []int{1, 4, 16, 64, 256}
	for _, count := range handlerCounts {
		b.Run(fmt.Sprintf("handlers=%d", count), func(b *testing.B) {
			m := newBenchmarkManager(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					dispatchHandlers(m, payload)
				}
			})
		})
	}
}
