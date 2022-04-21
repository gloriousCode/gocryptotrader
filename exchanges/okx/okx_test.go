package okx

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var ok Okx

func TestMain(m *testing.M) {
	ok.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Okx")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Okx); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return ok.ValidateAPICredentials(ok.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below
