package log

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) {
	for x := range s {
		output := getWriters(&s[x])
		err := configureSubLogger(strings.ToUpper(s[x].Name), s[x].Level, output)
		if err != nil {
			continue
		}
	}
}

func registerNewSubLogger(subLoggerName string) {
	subLogger := SubLogger{
		name:   strings.ToUpper(subLoggerName),
		output: os.Stdout,
	}

	subLogger.Levels = splitLevel(defaultLevels)
	rwm.Lock()
	subLoggers[subLoggerName] = &subLogger
	rwm.Unlock()
}

func configureSubLogger(logger, levels string, output io.Writer) error {
	logPtr := getSubLogger(logger)
	if logPtr == nil {
		return fmt.Errorf("logger %v not found", logger)
	}
	rwm.Lock()
	logPtr.output = output
	logPtr.Levels = splitLevel(levels)
	subLoggers[logger] = logPtr
	rwm.Unlock()
	return nil
}

func getSubLogger(s string) *SubLogger {
	rwm.RLock()
	defer rwm.RUnlock()
	if v, found := subLoggers[s]; found {
		return v
	}
	return nil
}
