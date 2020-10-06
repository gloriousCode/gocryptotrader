package log

import (
	"fmt"
	"log"
)

// Info takes a pointer SubLogger struct and string sends to newLogEvent
func Info(subLoggerName string, data string) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Info {
		return
	}

	displayError(logger.newLogEvent(data, logger.InfoHeader, subLogger.name, subLogger.output))
}

// Infoln takes a pointer SubLogger struct and interface sends to newLogEvent
func Infoln(subLoggerName string, v ...interface{}) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Info {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, subLogger.name, subLogger.output))
}

// Infof takes a pointer SubLogger struct, string & interface formats and sends to Info()
func Infof(subLoggerName string, data string, v ...interface{}) {
	Info(subLoggerName, fmt.Sprintf(data, v...))
}

// Debug takes a pointer SubLogger struct and string sends to multiwriter
func Debug(subLoggerName string, data string) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Debug {
		return
	}

	displayError(logger.newLogEvent(data, logger.DebugHeader, subLogger.name, subLogger.output))
}

// Debugln  takes a pointer SubLogger struct, string and interface sends to newLogEvent
func Debugln(subLoggerName string, v ...interface{}) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Debug {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, subLogger.name, subLogger.output))
}

// Debugf takes a pointer SubLogger struct, string & interface formats and sends to Info()
func Debugf(subLoggerName string, data string, v ...interface{}) {
	Debug(subLoggerName, fmt.Sprintf(data, v...))
}

// Warn takes a pointer SubLogger struct & string  and sends to newLogEvent()
func Warn(subLoggerName string, data string) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Warn {
		return
	}

	displayError(logger.newLogEvent(data, logger.WarnHeader, subLogger.name, subLogger.output))
}

// Warnln takes a pointer SubLogger struct & interface formats and sends to newLogEvent()
func Warnln(subLoggerName string, v ...interface{}) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Warn {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, subLogger.name, subLogger.output))
}

// Warnf takes a pointer SubLogger struct, string & interface formats and sends to Warn()
func Warnf(subLoggerName string, data string, v ...interface{}) {
	Warn(subLoggerName, fmt.Sprintf(data, v...))
}

// Error takes a pointer SubLogger struct & interface formats and sends to newLogEvent()
func Error(subLoggerName string, data ...interface{}) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprint(data...), logger.ErrorHeader, subLogger.name, subLogger.output))
}

// Errorln takes a pointer SubLogger struct, string & interface formats and sends to newLogEvent()
func Errorln(subLoggerName string, v ...interface{}) {
	if !enabled() {
		return
	}
	subLogger := getSubLogger(subLoggerName)
	if subLogger == nil || !subLogger.Error {
		return
	}

	displayError(logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, subLogger.name, subLogger.output))
}

// Errorf takes a pointer SubLogger struct, string & interface formats and sends to Debug()
func Errorf(subLoggerName string, data string, v ...interface{}) {
	Error(subLoggerName, fmt.Sprintf(data, v...))
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

func enabled() bool {
	rwm.RLock()
	defer rwm.RUnlock()
	if logConfig.Enabled == nil {
		return false
	}
	return *logConfig.Enabled
}
