package smsglobal

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestConnect(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.Connect()
	if err != nil {
		t.Error("SMSGlobal Connect() error", err)
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.PushEvent(base.Event{})
	if err != nil {
		t.Error("SMSGlobal PushEvent() error", err)
	}
}

func TestGetEnabledContacts(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
	v := s.GetEnabledContacts()
	if v != 1 {
		t.Error("SMSGlobal GetEnabledContacts() error")
	}
}

func TestGetContactByNumber(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
	_, err = s.GetContactByNumber("1231424")
	if err != nil {
		t.Error("SMSGlobal GetContactByNumber() error", err)
	}
	_, err = s.GetContactByNumber("basketball")
	if err == nil {
		t.Error("SMSGlobal GetContactByNumber() error")
	}
}

func TestGetContactByName(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
	_, err = s.GetContactByName("StyleGherkin")
	if err != nil {
		t.Error("SMSGlobal GetContactByName() error", err)
	}
	_, err = s.GetContactByName("blah")
	if err == nil {
		t.Error("SMSGlobal GetContactByName() error")
	}
}

func TestAddContact(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
	err = s.AddContact(Contact{Name: "bra", Number: "2876", Enabled: true})
	if err != nil {
		t.Error("SMSGlobal AddContact() error", err)
	}
	err = s.AddContact(Contact{Name: "StyleGherkin", Number: "1231424", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal AddContact() error")
	}
	err = s.AddContact(Contact{Name: "", Number: "", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal AddContact() error")
	}
}

func TestRemoveContact(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
	err = s.RemoveContact(Contact{Name: "StyleGherkin", Number: "1231424", Enabled: true})
	if err != nil {
		t.Error("SMSGlobal RemoveContact() error", err)
	}
	err = s.RemoveContact(Contact{Name: "frieda", Number: "243453", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal RemoveContact() Expected error")
	}
}

func TestSendMessageToAll(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.SendMessageToAll("Hello,World!")
	if err != nil {
		t.Error("SMSGlobal SendMessageToAll() error", err)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.SendMessage("1337", "Hello!")
	if err != nil {
		t.Error("SMSGlobal SendMessage() error", err)
	}
}
