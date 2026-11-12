package log

import (
	"sync"
	"sync/atomic"
)

// CustomLogHook is a function type for external log handling. It should return
// true if the library's internal logging system should be bypassed, or false
// if the library's internal logging system should be used.
type CustomLogHook func(header, subLoggerName string, a ...any) (bypassLibraryLogSystem bool)

var customLogHook CustomLogHook

// Observer receives a normalised log level and subsystem; message arguments are never supplied.
type Observer func(level, subLoggerName string)

var (
	observers      map[uint64]Observer
	observerSerial uint64
	observerMu     sync.Mutex
	observerState  atomic.Pointer[observerSnapshot]
)

// AddObserver registers a metadata callback and returns its independent registration ID.
// Observers run under the logging read lock and must not log or reconfigure loggers.
func AddObserver(observer Observer) uint64 {
	if observer == nil {
		return 0
	}
	observerMu.Lock()
	defer observerMu.Unlock()
	if observers == nil {
		observers = make(map[uint64]Observer)
	}
	observerSerial++
	observers[observerSerial] = observer
	snapshot := &observerSnapshot{callbacks: make([]Observer, 0, len(observers))}
	for _, callback := range observers {
		snapshot.callbacks = append(snapshot.callbacks, callback)
	}
	observerState.Store(snapshot)
	return observerSerial
}

// RemoveObserver excludes a registration from future snapshots; an in-flight callback may finish.
func RemoveObserver(id uint64) {
	observerMu.Lock()
	delete(observers, id)
	if len(observers) == 0 {
		observerState.Store(nil)
		observers = nil
	} else {
		snapshot := &observerSnapshot{callbacks: make([]Observer, 0, len(observers))}
		for _, callback := range observers {
			snapshot.callbacks = append(snapshot.callbacks, callback)
		}
		observerState.Store(snapshot)
	}
	observerMu.Unlock()
}

// SetCustomLogHook sets a custom log hook function that allows the complete
// bypass of the library's internal logging system. This is useful for
// implementing custom log handling.
func SetCustomLogHook(h CustomLogHook) {
	mu.Lock()
	customLogHook = h
	mu.Unlock()
}
