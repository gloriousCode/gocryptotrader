package log

import (
	"io"
	"sync"
)

const (
	// DefaultMaxFileSize for logger rotation file
	DefaultMaxFileSize int64 = 100

	timestampFormat = " 02/01/2006 15:04:05 "
	spacer          = " | "
	// Lovely strings
	infoStr                 = "INFO"
	infoFmt                 = "[" + infoStr + "]"
	warnStr                 = "WARN"
	warnFmt                 = "[" + warnStr + "]"
	debugStr                = "DEBUG"
	debugFmt                = "[" + debugStr + "]"
	errStr                  = "ERROR"
	errorFmt                = "[" + errStr + "]"
	defaultLevels           = infoStr + "|" + warnStr + "|" + debugStr + "|" + errStr
	configNotLoaded         = "config not loaded"
	nilConfigReceived       = "nil config received"
	filePathEmpty           = "log file path empty"
	missingLogSubSystemName = "missing showLogSystemName setting"
)

var (
	logger = &loggerDetails{}
	// fileLoggingConfiguredCorrectly flag set during config check if file logging meets requirements
	fileLoggingConfiguredCorrectly bool
	// logConfig holds configuration options for logger
	logConfig = &Config{}
	// rotate holds file rotation options for the logger
	rotate    = &Rotate{}
	eventPool = &sync.Pool{
		New: func() interface{} {
			return &event{
				data: make([]byte, 0, 80),
			}
		},
	}

	// filePath system path to store log files in
	filePath string
	// rwm read/write mutex for logger
	rwm sync.RWMutex
)

// Config holds configuration settings loaded from bot config
type Config struct {
	Enabled *bool `json:"enabled"`
	subLoggerConfig
	LoggerFileConfig *loggerFileConfig `json:"fileSettings,omitempty"`
	AdvancedSettings advancedSettings  `json:"advancedSettings"`
	SubLoggers       []subLoggerConfig `json:"subloggers,omitempty"`
}

type advancedSettings struct {
	ShowLogSystemName *bool   `json:"showLogSystemName"`
	Spacer            string  `json:"spacer"`
	TimeStampFormat   string  `json:"timeStampFormat"`
	Headers           headers `json:"headers"`
}

type headers struct {
	Info  string `json:"info"`
	Warn  string `json:"warn"`
	Debug string `json:"debug"`
	Error string `json:"error"`
}

// subLoggerConfig holds sub logger configuration settings loaded from bot config
type subLoggerConfig struct {
	Name   string `json:"name,omitempty"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

type loggerFileConfig struct {
	FileName string `json:"filename,omitempty"`
	Rotate   *bool  `json:"rotate,omitempty"`
	MaxSize  int64  `json:"maxsize,omitempty"`
}

// loggerDetails each instance of logger settings
type loggerDetails struct {
	ShowLogSystemName                                bool
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
}

// levels flags for each sub logger type
type Levels struct {
	Info, Debug, Warn, Error bool
}

// subLogger is a more specific logging level to help with categorising output
type subLoggerDetails struct {
	name string
	Levels
	output io.Writer
}

// event holds the data sent to the log and which multiwriter to send to
type event struct {
	data   []byte
	output io.Writer
}

type multiWriter struct {
	writers []io.Writer
	mu      sync.RWMutex
}
