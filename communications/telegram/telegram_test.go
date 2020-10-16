package telegram

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
)

const (
	testErrNotFound = "Not Found"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}

	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	tele := Telegram{}
	tele.Setup(&commsCfg)
	if tele.Name != "Telegram" || tele.Enabled || tele.Token != "testest" || tele.Verbose {
		t.Error("telegram Setup() error, unexpected setup values",
			tele.Name,
			tele.Enabled,
			tele.Token,
			tele.Verbose)
	}
}

func TestConnect(t *testing.T) {
	tele := Telegram{}
	err := tele.Connect()
	if err == nil {
		t.Error("telegram Connect() error")
	}
}

func TestPushEvent(t *testing.T) {
	tele := Telegram{}
	err := tele.PushEvent(base.Event{})
	if err != nil {
		t.Error("telegram PushEvent() error", err)
	}
	tele.AuthorisedClients = append(tele.AuthorisedClients, 1337)
	err = tele.PushEvent(base.Event{})
	if err != nil && err.Error() != testErrNotFound {
		t.Errorf("telegram PushEvent() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestHandleMessages(t *testing.T) {
	t.Parallel()
	tele := Telegram{}
	chatID := int64(1337)
	err := tele.HandleMessages(cmdHelp, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = tele.HandleMessages(cmdStart, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = tele.HandleMessages(cmdStatus, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = tele.HandleMessages(cmdSettings, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = tele.HandleMessages("Not a command", chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestGetUpdates(t *testing.T) {
	tele := Telegram{}
	t.Parallel()
	_, err := tele.GetUpdates()
	if err != nil {
		t.Error("telegram GetUpdates() error", err)
	}
}

func TestTestConnection(t *testing.T) {
	tele := Telegram{}
	t.Parallel()
	err := tele.TestConnection()
	if err.Error() != testErrNotFound {
		t.Errorf("telegram TestConnection() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestSendMessage(t *testing.T) {
	tele := Telegram{}
	t.Parallel()
	err := tele.SendMessage("Test message", int64(1337))
	if err.Error() != testErrNotFound {
		t.Errorf("telegram SendMessage() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	tele := Telegram{}
	t.Parallel()
	err := tele.SendHTTPRequest("0.0.0.0", nil, nil)
	if err == nil {
		t.Error("telegram SendHTTPRequest() error")
	}
}
