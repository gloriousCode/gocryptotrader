package log

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var errUnsupportedLoggerBackend = errors.New("unsupported logger backend")

// setupSubLoggerBackend configures a sublogger backend implementation.
// Note: Calling function must have mutex lock in place.
func setupSubLoggerBackend(sl *SubLogger) error {
	if sl == nil {
		return nil
	}
	sl.zapLogger = nil
	sl.buildPrefixes()

	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:          "ts",
		LevelKey:         "",
		NameKey:          "",
		MessageKey:       "message",
		LineEnding:       zapcore.DefaultLineEnding,
		ConsoleSeparator: logger.Spacer,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(logger.TimestampFormat))
		},
	})

	writeSyncers := make([]zapcore.WriteSyncer, 0, len(sl.output.writers))
	for i := range sl.output.writers {
		writeSyncers = append(writeSyncers, zapcore.AddSync(sl.output.writers[i]))
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writeSyncers...),
		zapcore.DebugLevel,
	)
	sl.zapLogger = zap.New(core)
	return nil
}

// normaliseLoggerBackend sanitises and validates logger backend settings.
func normaliseLoggerBackend(backend string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(backend)) {
	case zapBackend:
		return zapBackend, nil
	case "":
		return zapBackend, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedLoggerBackend, backend)
	}
}
