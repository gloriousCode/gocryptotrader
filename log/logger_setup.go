package log

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

// register all loggers at package init()
func init() {
	registerNewSubLogger(Global)
	registerNewSubLogger(ConnectionMgr)
	registerNewSubLogger(CommunicationMgr)
	registerNewSubLogger(ConfigMgr)
	registerNewSubLogger(DatabaseMgr)
	registerNewSubLogger(OrderMgr)
	registerNewSubLogger(PortfolioMgr)
	registerNewSubLogger(SyncMgr)
	registerNewSubLogger(TimeMgr)
	registerNewSubLogger(GCTScriptMgr)
	registerNewSubLogger(WebsocketMgr)
	registerNewSubLogger(EventMgr)
	registerNewSubLogger(DispatchMgr)
	registerNewSubLogger(RequestSys)
	registerNewSubLogger(ExchangeSys)
	registerNewSubLogger(GRPCSys)
	registerNewSubLogger(RESTSys)
	registerNewSubLogger(Ticker)
	registerNewSubLogger(OrderBook)
}

func getWriters(s *SubLoggerConfig) io.Writer {
	mw := MultiWriter()
	m := mw.(*multiWriter)
	outputWriters := strings.Split(s.Output, "|")
	for x := range outputWriters {
		switch outputWriters[x] {
		case "stdout", "console":
			m.Add(os.Stdout)
		case "stderr":
			m.Add(os.Stderr)
		case "file":
			if isLogConfiguredCorrectly() {
				m.Add(rotate)
			}
		default:
			m.Add(ioutil.Discard)
		}
	}
	return m
}

// GenDefaultSettings return struct with known sane/working logger settings
func GenDefaultSettings() (log Config) {
	log = Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Level:  defaultLevels,
			Output: "console",
		},
		LoggerFileConfig: &loggerFileConfig{
			FileName: "log.txt",
			Rotate:   convert.BoolPtr(false),
			MaxSize:  0,
		},
		AdvancedSettings: advancedSettings{
			ShowLogSystemName: convert.BoolPtr(false),
			Spacer:            spacer,
			TimeStampFormat:   timestampFormat,
			Headers: headers{
				Info:  infoFmt,
				Warn:  warnFmt,
				Debug: debugFmt,
				Error: errorFmt,
			},
		},
	}
	return
}

// SetupGlobalLogger setup the global loggers with the default global config values
func SetupGlobalLogger() {
	isCorrect := isLogConfiguredCorrectly()
	rwm.Lock()
	if isCorrect {
		rotate = &Rotate{
			fileName:        logConfig.LoggerFileConfig.FileName,
			maxSize:         logConfig.LoggerFileConfig.MaxSize,
			rotationEnabled: logConfig.LoggerFileConfig.Rotate,
		}
	}
	for x := range subLoggers {
		subLoggers[x].Levels = splitLevel(logConfig.Level)
		subLoggers[x].output = getWriters(&logConfig.SubLoggerConfig)
	}
	rwm.Unlock()
	setLogger(newLogger(logConfig))
}

func splitLevel(level string) (l Levels) {
	enabledLevels := strings.Split(level, "|")
	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
		case debugStr:
			l.Debug = true
		case infoStr:
			l.Info = true
		case warnStr:
			l.Warn = true
		case errStr:
			l.Error = true
		}
	}
	return
}
