package log

import (
	"io"
	"testing"
)

// newBenchmarkSubLogger creates a benchmark logger that works across
// historical and current logger implementations.
func newBenchmarkSubLogger(b *testing.B) *SubLogger {
	b.Helper()

	sl := &SubLogger{
		name:    "BENCH",
		botName: "bench",
	}
	if err := sl.setOutput(&multiWriterHolder{
		writers: []io.Writer{io.Discard},
	}); err != nil {
		b.Fatalf("setOutput() error: %v", err)
	}
	sl.setLevels(Levels{Info: true, Debug: true, Warn: true, Error: true})
	return sl
}

func BenchmarkInfolnDiscard(b *testing.B) {
	mu.Lock()
	originalLogger := logger
	logger.BypassJobChannelFilledWarning = true
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		logger = originalLogger
		mu.Unlock()
	})

	sl := newBenchmarkSubLogger(b)

	b.ResetTimer()
	for b.Loop() {
		Infoln(sl, "Hello this is an infoln benchmark")
	}
}

func BenchmarkInfofDiscard(b *testing.B) {
	mu.Lock()
	originalLogger := logger
	logger.BypassJobChannelFilledWarning = true
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		logger = originalLogger
		mu.Unlock()
	})

	sl := newBenchmarkSubLogger(b)

	b.ResetTimer()
	for n := range b.N {
		Infof(sl, "Hello this is an infof benchmark %v %v %v", n, 1, 2)
	}
}

func BenchmarkInfofDiscardWithCustomHook(b *testing.B) {
	mu.Lock()
	originalLogger := logger
	logger.BypassJobChannelFilledWarning = true
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		logger = originalLogger
		mu.Unlock()
	})

	sl := newBenchmarkSubLogger(b)

	customLogHook = func(_, _ string, _ ...any) bool {
		return false
	}
	b.Cleanup(func() {
		customLogHook = nil
	})

	b.ResetTimer()
	for n := range b.N {
		Infof(sl, "Hello this is an infof benchmark %v %v %v", n, 1, 2)
	}
}
