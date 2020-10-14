package log

import (
	"io"
	"sync"
)

const (
	timestampFormat = " 02/01/2006 15:04:05 "
	spacer          = " | "
	// DefaultMaxFileSize for logger rotation file
	DefaultMaxFileSize int64 = 100
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
	nilConfigReceived       = "nil config received"
	filePathEmpty           = "log file path empty"
	missingLogSubSystemName = "missing showLogSystemName setting"
	missingEnabled          = "missing enabled setting"
)

var (
	logger = &Logger{}
	// fileLoggingConfiguredCorrectly flag set during config check if file logging meets requirements
	fileLoggingConfiguredCorrectly bool
	// logConfig holds configuration options for logger
	logConfig = &Config{}
	// rotate holds file rotation options for the logger
	rotate    = &Rotate{}
	eventPool = &sync.Pool{
		New: func() interface{} {
			return &Event{
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
	SubLoggerConfig
	LoggerFileConfig *loggerFileConfig `json:"fileSettings,omitempty"`
	AdvancedSettings advancedSettings  `json:"advancedSettings"`
	SubLoggers       []SubLoggerConfig `json:"subloggers,omitempty"`
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

// SubLoggerConfig holds sub logger configuration settings loaded from bot config
type SubLoggerConfig struct {
	Name   string `json:"name,omitempty"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

type loggerFileConfig struct {
	FileName string `json:"filename,omitempty"`
	Rotate   *bool  `json:"rotate,omitempty"`
	MaxSize  int64  `json:"maxsize,omitempty"`
}

// Logger each instance of logger settings
type Logger struct {
	ShowLogSystemName                                bool
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
}

// Levels flags for each sub logger type
type Levels struct {
	Info, Debug, Warn, Error bool
}

type SubLogger struct {
	name string
	Levels
	output io.Writer
}

// Event holds the data sent to the log and which multiwriter to send to
type Event struct {
	data   []byte
	output io.Writer
}

type multiWriter struct {
	writers []io.Writer
	mu      sync.RWMutex
}
