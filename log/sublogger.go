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

func registerNewSubLogger(logger string) *SubLogger {
	temp := SubLogger{
		name:   strings.ToUpper(logger),
		output: os.Stdout,
	}

	temp.Levels = splitLevel("INFO|WARN|DEBUG|ERROR")
	rwm.Lock()
	subLoggers[logger] = &temp
	rwm.Unlock()
	return &temp
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
