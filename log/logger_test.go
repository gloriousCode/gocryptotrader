package log

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestMain(m *testing.M) {
	setupTestLoggers()
	os.Exit(m.Run())
}

func setupTestLoggers() {
	logTest := Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Output: "console",
			Level:  defaultLevels,
		},
		AdvancedSettings: advancedSettings{
			ShowLogSystemName: convert.BoolPtr(true),
			Spacer:            spacer,
			TimeStampFormat:   timestampFormat,
			Headers: headers{
				Info:  infoFmt,
				Warn:  warnFmt,
				Debug: debugFmt,
				Error: errorFmt,
			},
		},
		SubLoggers: []SubLoggerConfig{
			{
				Name:   "TEST",
				Level:  defaultLevels,
				Output: "stdout",
			}},
	}
	SetConfig(&logTest)
	SetupGlobalLogger()
	SetupSubLoggers(logTest.SubLoggers)
}

func SetupDisabled() {
	logTest := Config{
		Enabled: convert.BoolPtr(false),
	}
	SetConfig(&logTest)
	SetupGlobalLogger()
	SetupSubLoggers(logTest.SubLoggers)
}

func BenchmarkInfo(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func TestAddWriter(t *testing.T) {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	m.Add(ioutil.Discard)
	m.Add(os.Stdin)
	m.Add(os.Stdout)

	total := len(m.writers)

	if total != 3 {
		t.Errorf("expected m.Writers to be 3 %v", total)
	}
}

func TestRemoveWriter(t *testing.T) {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	m.Add(ioutil.Discard)
	m.Add(os.Stdin)
	m.Add(os.Stdout)

	total := len(m.writers)

	m.Remove(os.Stdin)
	m.Remove(os.Stdout)

	if len(m.writers) != total-2 {
		t.Errorf("expected m.Writers to be %v got %v", total-2, len(m.writers))
	}
}

func TestLevel(t *testing.T) {
	_, err := Level(Global)
	if err != nil {
		t.Errorf("Failed to get log %s levels skipping", err)
	}

	_, err = Level("totallyinvalidlogger")
	if err == nil {
		t.Error("Expected error on invalid logger")
	}
}

func TestSetLevel(t *testing.T) {
	newLevel, err := SetLevel(Global, errStr)
	if err != nil {
		t.Skipf("Failed to get log %s levels skipping", err)
	}

	if newLevel.Info || newLevel.Debug || newLevel.Warn {
		t.Error("failed to set level correctly")
	}

	if !newLevel.Error {
		t.Error("failed to set level correctly")
	}

	_, err = SetLevel("abc12345556665", errStr)
	if err == nil {
		t.Error("SetLevel() Should return error on invalid logger")
	}
}

func TestValidSubLogger(t *testing.T) {
	logPtr := getSubLogger(Global)

	if logPtr == nil {
		t.Error("getSubLogger() should return a pointer and not nil")
	}
}

func TestCloseLogger(t *testing.T) {
	err := CloseLogger()
	if err != nil {
		t.Errorf("CloseLogger() failed %v", err)
	}
}

func TestConfigureSubLogger(t *testing.T) {
	err := configureSubLogger(Global, infoStr, os.Stdin)
	if err != nil {
		t.Skipf("configureSubLogger() returned unexpected error %v", err)
	}
	global := getSubLogger(Global)
	if (global.Levels != Levels{
		Info:  true,
		Debug: false,
	}) {
		t.Error("configureSubLogger() incorrectly configure SubLogger")
	}
	if global.name != Global {
		t.Error("configureSubLogger() Failed to uppercase name")
	}
}

func TestSplitLevel(t *testing.T) {
	levelsInfoDebug := splitLevel("INFO|DEBUG")

	expected := Levels{
		Info:  true,
		Debug: true,
		Warn:  false,
		Error: false,
	}

	if levelsInfoDebug != expected {
		t.Errorf("splitLevel() returned invalid data expected: %+v got: %+v", expected, levelsInfoDebug)
	}
}

func BenchmarkInfoDisabled(b *testing.B) {
	SetupDisabled()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func BenchmarkInfof(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof(Global, "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

func BenchmarkInfoln(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln(Global, "Hello this is an infoln benchmark")
	}
}

func TestNewLogEvent(t *testing.T) {
	w := &bytes.Buffer{}
	err := logger.newLogEvent("out", "header", "SUBLOGGER", w)
	if err != nil {
		t.Fatal(err)
	}
	if w.String() == "" {
		t.Error("newLogEvent() failed expected output got empty string")
	}

	err = logger.newLogEvent("out", "header", "SUBLOGGER", nil)
	if err == nil {
		t.Error("Error expected with output is set to nil")
	}
}

func TestInfo(t *testing.T) {
	w := &bytes.Buffer{}
	tempSL := SubLogger{
		"TESTYMCTESTALOT",
		splitLevel(defaultLevels),
		w,
	}
	rwm.Lock()
	subLoggers["TESTYMCTESTALOT"] = &tempSL
	rwm.Unlock()
	Info("TESTYMCTESTALOT", "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}

	tempSL.output = nil
	w.Reset()

	_, err := SetLevel("TESTYMCTESTALOT", infoStr)
	if err != nil {
		t.Error(err)
	}

	Debug("TESTYMCTESTALOT", "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but Debug wrote to output")
	}
}

func TestSubLoggerName(t *testing.T) {
	w := &bytes.Buffer{}
	registerNewSubLogger("sublogger")
	err := logger.newLogEvent("out", "header", "SUBLOGGER", w)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Expected SUBLOGGER in output")
	}

	logger.ShowLogSystemName = false
	w.Reset()
	err = logger.newLogEvent("out", "header", "SUBLOGGER", w)
	if err != nil {
		t.Error(err)
	}
	if strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Unexpected SUBLOGGER in output")
	}
}
