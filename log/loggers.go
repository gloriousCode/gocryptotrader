package log

import (
	"fmt"
	"io"
	"log"
)

// Info takes a pointer subLogger struct and string sends to newLogEvent
func Info(sl *subLogger, data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.info {
		return
	}

	displayError(logger.newLogEvent(data, logger.InfoHeader, fields.name, fields.output))
}

// Infoln takes a pointer subLogger struct and interface sends to newLogEvent
func Infoln(sl *subLogger, v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.info {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, fields.name, fields.output))
}

// Infof takes a pointer subLogger struct, string & interface formats and sends to Info()
func Infof(sl *subLogger, data string, v ...interface{}) {
	Info(sl, fmt.Sprintf(data, v...))
}

// Debug takes a pointer subLogger struct and string sends to multiwriter
func Debug(sl *subLogger, data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.debug {
		return
	}

	displayError(logger.newLogEvent(data, logger.DebugHeader, fields.name, fields.output))
}

// Debugln  takes a pointer subLogger struct, string and interface sends to newLogEvent
func Debugln(sl *subLogger, v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.debug {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, fields.name, fields.output))
}

// Debugf takes a pointer subLogger struct, string & interface formats and sends to Info()
func Debugf(sl *subLogger, data string, v ...interface{}) {
	Debug(sl, fmt.Sprintf(data, v...))
}

// Warn takes a pointer subLogger struct & string  and sends to newLogEvent()
func Warn(sl *subLogger, data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.warn {
		return
	}

	displayError(logger.newLogEvent(data, logger.WarnHeader, fields.name, fields.output))
}

// Warnln takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func Warnln(sl *subLogger, v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.warn {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, fields.name, fields.output))
}

// Warnf takes a pointer subLogger struct, string & interface formats and sends to Warn()
func Warnf(sl *subLogger, data string, v ...interface{}) {
	Warn(sl, fmt.Sprintf(data, v...))
}

// Error takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func Error(sl *subLogger, data ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprint(data...), logger.ErrorHeader, fields.name, fields.output))
}

// Errorln takes a pointer subLogger struct, string & interface formats and sends to newLogEvent()
func Errorln(sl *subLogger, v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, fields.name, fields.output))
}

// Errorf takes a pointer subLogger struct, string & interface formats and sends to Debug()
func Errorf(sl *subLogger, data string, v ...interface{}) {
	Error(sl, fmt.Sprintf(data, v...))
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

func enabled() bool {
	RWM.Lock()
	defer RWM.Unlock()
	if GlobalLogConfig == nil || GlobalLogConfig.Enabled == nil {
		return false
	}
	if *GlobalLogConfig.Enabled {
		return true
	}
	return false
}

func getFields(sl *subLogger) *logFields {
	if !enabled() {
		return nil
	}
	RWM.RLock()
	defer RWM.RUnlock()
	if sl == nil {
		return nil
	}
	return &logFields{
		info:   sl.Info,
		warn:   sl.Warn,
		debug:  sl.Debug,
		error:  sl.Error,
		name:   sl.name,
		output: sl.output,
	}
}

type logFields struct {
	info   bool
	warn   bool
	debug  bool
	error  bool
	name   string
	output io.Writer
}
