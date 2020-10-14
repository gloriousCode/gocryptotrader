package log

import (
	"errors"
	"fmt"
	"io"
	"time"
)

func newLogger(c *Config) *Logger {
	return &Logger{
		Timestamp:         c.AdvancedSettings.TimeStampFormat,
		Spacer:            c.AdvancedSettings.Spacer,
		ErrorHeader:       c.AdvancedSettings.Headers.Error,
		InfoHeader:        c.AdvancedSettings.Headers.Info,
		WarnHeader:        c.AdvancedSettings.Headers.Warn,
		DebugHeader:       c.AdvancedSettings.Headers.Debug,
		ShowLogSystemName: *c.AdvancedSettings.ShowLogSystemName,
	}
}

func (l *Logger) newLogEvent(data, header, slName string, w io.Writer) error {
	if w == nil {
		return errors.New("io.Writer not set")
	}

	e := eventPool.Get().(*Event)
	e.output = w
	e.data = append(e.data, []byte(header)...)
	if l.ShowLogSystemName {
		e.data = append(e.data, l.Spacer...)
		e.data = append(e.data, slName...)
	}
	e.data = append(e.data, l.Spacer...)
	if l.Timestamp != "" {
		e.data = time.Now().AppendFormat(e.data, l.Timestamp)
	}
	e.data = append(e.data, l.Spacer...)
	e.data = append(e.data, []byte(data)...)
	if data == "" || data[len(data)-1] != '\n' {
		e.data = append(e.data, '\n')
	}
	_, err := e.output.Write(e.data)

	e.data = e.data[:0]
	eventPool.Put(e)

	return err
}

// CloseLogger is called on shutdown of application
func CloseLogger() error {
	err := rotate.Close()
	if err != nil {
		return err
	}
	return nil
}

// Level retries the current sublogger levels
func Level(s string) (*Levels, error) {
	logger := getSubLogger(s)
	if logger == nil {
		return nil, fmt.Errorf("logger %v not found", s)
	}

	return &logger.Levels, nil
}

// SetLevel sets sublogger levels
func SetLevel(s, level string) (*Levels, error) {
	logger := getSubLogger(s)
	if logger == nil {
		return nil, fmt.Errorf("logger %v not found", s)
	}
	logger.Levels = splitLevel(level)

	return &logger.Levels, nil
}

func setLogger(l *Logger) {
	rwm.Lock()
	logger = l
	rwm.Unlock()
}

func isFileLoggingSetup() bool {
	rwm.RLock()
	defer rwm.RUnlock()
	b := fileLoggingConfiguredCorrectly
	return b
}

// SetLogConfiguredCorrectly sets whether the logger
// has been configured correctly
func SetLogConfiguredCorrectly(b bool) {
	rwm.Lock()
	fileLoggingConfiguredCorrectly = b
	rwm.Unlock()
}

// SetConfig sets the config
func SetConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New(nilConfigReceived)
	}
	if cfg.Enabled == nil {
		return errors.New(missingEnabled)
	}
	if cfg.AdvancedSettings.ShowLogSystemName == nil {
		return errors.New(missingLogSubSystemName)
	}
	rwm.Lock()
	logConfig = cfg
	rwm.Unlock()
	return nil
}
