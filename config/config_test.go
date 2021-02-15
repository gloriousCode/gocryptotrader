package config

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/ntpclient"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	// Default number of enabled exchanges. Modify this whenever an exchange is
	// added or removed
	defaultEnabledExchanges = 28
	testFakeExchangeName    = "Stampbit"
	testPair                = "BTC-USD"
	testString              = "test"
)

func TestGetNonExistentDefaultFilePathDoesNotCreateDefaultDir(t *testing.T) {
	dir := common.GetDefaultDataDir(runtime.GOOS)
	if file.Exists(dir) {
		t.Skip("The default directory already exists before running the test")
	}
	GetFilePath("")
	if file.Exists(dir) {
		t.Fatalf("The target directory was created in %s", dir)
	}
}

func TestGetCurrencyConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCurrencyConfig LoadConfig error", err)
	}
	_ = cfg.GetCurrencyConfig()
}

func TestGetClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Fatal("GetExchangeBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetClientBankAccounts("Kraken", "USD")
	if err != nil {
		t.Error("GetExchangeBankAccounts error", err)
	}
	_, err = cfg.GetClientBankAccounts("noob exchange", "USD")
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestGetExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetExchangeBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Bitfinex", "", "USD")
	if err != nil {
		t.Error("GetExchangeBankAccounts error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Not an exchange", "", "Not a currency")
	if err == nil {
		t.Error("GetExchangeBankAccounts, no error returned for invalid exchange")
	}
}

func TestCheckBankAccountConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetExchangeBankAccounts LoadConfig error", err)
	}

	cfg.BankAccounts[0].Enabled = true
	cfg.CheckBankAccountConfig()
}

func TestUpdateExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateExchangeBankAccounts LoadConfig error", err)
	}

	b := []banking.Account{{Enabled: false}}
	err = cfg.UpdateExchangeBankAccounts("Bitfinex", b)
	if err != nil {
		t.Error("UpdateExchangeBankAccounts error", err)
	}
	var count int
	for _, exch := range cfg.Exchanges {
		if exch.Name == "Bitfinex" {
			if !exch.BankAccounts[0].Enabled {
				count++
			}
		}
	}
	if count != 1 {
		t.Error("UpdateExchangeBankAccounts error")
	}

	err = cfg.UpdateExchangeBankAccounts("Not an exchange", b)
	if err == nil {
		t.Error("UpdateExchangeBankAccounts, no error returned for invalid exchange")
	}
}

func TestUpdateClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateClientBankAccounts LoadConfig error", err)
	}
	b := banking.Account{Enabled: false, BankName: testString, AccountNumber: "0234"}
	err = cfg.UpdateClientBankAccounts(&b)
	if err != nil {
		t.Error("UpdateClientBankAccounts error", err)
	}

	err = cfg.UpdateClientBankAccounts(&banking.Account{})
	if err == nil {
		t.Error("UpdateClientBankAccounts error")
	}

	var count int
	for _, bank := range cfg.BankAccounts {
		if bank.BankName == b.BankName {
			if !bank.Enabled {
				count++
			}
		}
	}
	if count != 1 {
		t.Error("UpdateClientBankAccounts error")
	}
}

func TestCheckClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("CheckClientBankAccounts LoadConfig error", err)
	}

	cfg.BankAccounts = nil
	cfg.CheckClientBankAccounts()
	if len(cfg.BankAccounts) == 0 {
		t.Error("CheckClientBankAccounts error:", err)
	}

	cfg.BankAccounts = nil
	cfg.BankAccounts = []banking.Account{
		{
			Enabled: true,
		},
	}

	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	b := banking.Account{
		Enabled:             true,
		BankName:            "Commonwealth Bank of Awesome",
		BankAddress:         "123 Fake Street",
		BankPostalCode:      "1337",
		BankPostalCity:      "Satoshiville",
		BankCountry:         "Genesis",
		AccountName:         "Satoshi Nakamoto",
		AccountNumber:       "1231006505",
		SupportedCurrencies: "USD",
	}
	cfg.BankAccounts = []banking.Account{b}
	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled ||
		cfg.BankAccounts[0].SupportedExchanges != "ALL" {
		t.Error("unexpected result")
	}

	// AU based bank, with no BSB number (required for domestic and international
	// transfers)
	b.SupportedCurrencies = "AUD"
	b.SWIFTCode = "BACXSI22"
	cfg.BankAccounts = []banking.Account{b}
	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	// Valid AU bank
	b.BSBNumber = "061337"
	cfg.BankAccounts = []banking.Account{b}
	cfg.CheckClientBankAccounts()
	if !cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	// Valid SWIFT/IBAN compliant bank
	b.Enabled = true
	b.IBAN = "SI56290000170073837"
	b.SWIFTCode = "BACXSI22"
	cfg.BankAccounts = []banking.Account{b}
	cfg.CheckClientBankAccounts()
	if !cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}
}

func TestPurgeExchangeCredentials(t *testing.T) {
	t.Parallel()
	var c Config
	c.Exchanges = []ExchangeConfig{
		{
			Name: testString,
			API: APIConfig{
				AuthenticatedSupport:          true,
				AuthenticatedWebsocketSupport: true,
				CredentialsValidator: &APICredentialsValidatorConfig{
					RequiresKey:      true,
					RequiresSecret:   true,
					RequiresClientID: true,
				},
				Credentials: APICredentialsConfig{
					Key:       "asdf123",
					Secret:    "secretp4ssw0rd",
					ClientID:  "1337",
					OTPSecret: "otp",
					PEMKey:    "aaa",
				},
			},
		},
		{
			Name: "test123",
			API: APIConfig{
				CredentialsValidator: &APICredentialsValidatorConfig{
					RequiresKey: true,
				},
				Credentials: APICredentialsConfig{
					Key:    "asdf",
					Secret: DefaultAPISecret,
				},
			},
		},
	}

	c.PurgeExchangeAPICredentials()

	exchCfg, err := c.GetExchangeConfig(testString)
	if err != nil {
		t.Error(err)
	}

	if exchCfg.API.Credentials.Key != DefaultAPIKey &&
		exchCfg.API.Credentials.ClientID != DefaultAPIClientID &&
		exchCfg.API.Credentials.Secret != DefaultAPISecret &&
		exchCfg.API.Credentials.OTPSecret != "" &&
		exchCfg.API.Credentials.PEMKey != "" {
		t.Error("unexpected values")
	}

	exchCfg, err = c.GetExchangeConfig("test123")
	if err != nil {
		t.Error(err)
	}

	if exchCfg.API.Credentials.Key != "asdf" {
		t.Error("unexpected values")
	}
}

func TestGetCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCommunicationsConfig LoadConfig error", err)
	}
	_ = cfg.GetCommunicationsConfig()
}

func TestUpdateCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateCommunicationsConfig LoadConfig error", err)
	}
	cfg.UpdateCommunicationsConfig(&CommunicationsConfig{SlackConfig: SlackConfig{Name: testString}})
	if cfg.Communications.SlackConfig.Name != testString {
		t.Error("UpdateCommunicationsConfig LoadConfig error")
	}
}

func TestGetCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCryptocurrencyProviderConfig LoadConfig error", err)
	}
	_ = cfg.GetCryptocurrencyProviderConfig()
}

func TestUpdateCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateCryptocurrencyProviderConfig LoadConfig error", err)
	}

	orig := cfg.GetCryptocurrencyProviderConfig()
	cfg.UpdateCryptocurrencyProviderConfig(CryptocurrencyProvider{Name: "SERIOUS TESTING PROCEDURE!"})
	if cfg.Currency.CryptocurrencyProvider.Name != "SERIOUS TESTING PROCEDURE!" {
		t.Error("UpdateCurrencyProviderConfig LoadConfig error")
	}

	cfg.UpdateCryptocurrencyProviderConfig(orig)
}

func TestCheckCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("CheckCommunicationsConfig LoadConfig error", err)
	}

	cfg.Communications = CommunicationsConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Name != "Slack" ||
		cfg.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		cfg.Communications.SMTPConfig.Name != "SMTP" ||
		cfg.Communications.TelegramConfig.Name != "Telegram" {
		t.Error("CheckCommunicationsConfig unexpected data:",
			cfg.Communications)
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Password != testString {
		t.Error("CheckCommunicationsConfig error:", err)
	}

	cfg.SMS.Contacts = append(cfg.SMS.Contacts, SMSContact{
		Name:    "Bobby",
		Number:  "4321",
		Enabled: false,
	})
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Contacts[0].Name != "Bobby" {
		t.Error("CheckCommunicationsConfig error:", err)
	}

	cfg.Communications.SMSGlobalConfig.From = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.From != cfg.Name {
		t.Error("CheckCommunicationsConfig From value should have been set to the config name")
	}

	cfg.Communications.SMSGlobalConfig.From = "aaaaaaaaaaaaaaaaaaa"
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.From != "aaaaaaaaaaa" {
		t.Error("CheckCommunicationsConfig From value should have been trimmed to 11 characters")
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.SMS != nil {
		t.Error("CheckCommunicationsConfig unexpected data:",
			cfg.SMS)
	}

	cfg.Communications.SlackConfig.Name = "NOT Slack"
	cfg.CheckCommunicationsConfig()

	cfg.Communications.SlackConfig.Name = "Slack"
	cfg.Communications.SlackConfig.Enabled = true
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig Slack is enabled when it shouldn't be.")
	}

	cfg.Communications.SlackConfig.Enabled = false
	cfg.Communications.SMSGlobalConfig.Enabled = true
	cfg.Communications.SMSGlobalConfig.Password = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig SMSGlobal is enabled when it shouldn't be.")
	}

	cfg.Communications.SMSGlobalConfig.Enabled = false
	cfg.Communications.SMTPConfig.Enabled = true
	cfg.Communications.SMTPConfig.AccountPassword = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig SMTPConfig is enabled when it shouldn't be.")
	}

	cfg.Communications.SMTPConfig.Enabled = false
	cfg.Communications.TelegramConfig.Enabled = true
	cfg.Communications.TelegramConfig.VerificationToken = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.TelegramConfig.Enabled {
		t.Error("CheckCommunicationsConfig TelegramConfig is enabled when it shouldn't be.")
	}
}

func TestGetExchangeAssetTypes(t *testing.T) {
	t.Parallel()
	var c Config
	_, err := c.GetExchangeAssetTypes("void")
	if err == nil {
		t.Error("err should have been thrown on a non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot:    new(currency.PairStore),
					asset.Futures: new(currency.PairStore),
				},
			},
		},
	)

	var assets asset.Items
	assets, err = c.GetExchangeAssetTypes(testFakeExchangeName)
	if err != nil {
		t.Error(err)
	}

	if !assets.Contains(asset.Spot) || !assets.Contains(asset.Futures) {
		t.Error("unexpected results")
	}

	c.Exchanges[0].CurrencyPairs = nil
	_, err = c.GetExchangeAssetTypes(testFakeExchangeName)
	if err == nil {
		t.Error("Expected error from nil currency pair")
	}
}

func TestSupportsExchangeAssetType(t *testing.T) {
	t.Parallel()
	var c Config
	err := c.SupportsExchangeAssetType("void", asset.Spot)
	if err == nil {
		t.Error("Expected error for non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: new(currency.PairStore),
				},
			},
		},
	)

	err = c.SupportsExchangeAssetType(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	err = c.SupportsExchangeAssetType(testFakeExchangeName, "asdf")
	if err == nil {
		t.Error("Expected error from invalid asset item")
	}

	c.Exchanges[0].CurrencyPairs = nil
	err = c.SupportsExchangeAssetType(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}
}

func TestSetPairs(t *testing.T) {
	t.Parallel()

	var c Config
	pairs := currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	err := c.SetPairs("asdf", asset.Spot, true, nil)
	if err == nil {
		t.Error("Expected error from nil pairs")
	}

	err = c.SetPairs("asdf", asset.Spot, true, pairs)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	err = c.SetPairs(testFakeExchangeName, asset.Index, true, pairs)
	if err == nil {
		t.Error("Expected error from non initialised pair manager")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: new(currency.PairStore),
		},
	}

	err = c.SetPairs(testFakeExchangeName, asset.Index, true, pairs)
	if err == nil {
		t.Error("Expected error from non supported asset type")
	}

	err = c.SetPairs(testFakeExchangeName, asset.Spot, true, pairs)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyPairConfig(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetCurrencyPairConfig("asdfg", asset.Spot)
	if err == nil {
		t.Error("Expected error with non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	_, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Index)
	if err == nil {
		t.Error("Expected error with nil currency pair store")
	}

	pm := &currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				RequestFormat: &currency.PairFormat{
					Uppercase: false,
					Delimiter: "_",
				},
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
					Delimiter: "~",
				},
			},
		},
	}

	c.Exchanges[0].CurrencyPairs = pm
	_, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Index)
	if err == nil {
		t.Error("Expected error with unsupported asset")
	}

	var p *currency.PairStore
	p, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if p.RequestFormat.Delimiter != "_" ||
		p.RequestFormat.Uppercase ||
		!p.ConfigFormat.Uppercase ||
		p.ConfigFormat.Delimiter != "~" {
		t.Error("unexpected values")
	}
}

func TestCheckPairConfigFormats(t *testing.T) {
	var c Config
	if err := c.CheckPairConfigFormats("non-existent"); err == nil {
		t.Error("non-existent exchange should throw an error")
	}

	// Test nil pair store
	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("nil pair store should return an error")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot:    {},
			asset.Futures: {},
		},
	}
	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("error cannot be nil")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				RequestFormat: &currency.PairFormat{},
				ConfigFormat:  &currency.PairFormat{},
			},
			asset.Futures: {
				RequestFormat: &currency.PairFormat{},
				ConfigFormat:  &currency.PairFormat{},
			},
		},
	}
	if err := c.CheckPairConfigFormats(testFakeExchangeName); err != nil {
		t.Error("nil pairs should be okay to continue")
	}
	avail, err := currency.NewPairDelimiter(testPair, "-")
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := currency.NewPairDelimiter("BTC~USD", "~")
	if err != nil {
		t.Fatal(err)
	}
	// Test having a pair index and delimiter set at the same time throws an error
	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			RequestFormat: &currency.PairFormat{
				Uppercase: false,
				Delimiter: "_",
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "~",
				Index:     "USD",
			},
			Available: currency.Pairs{
				avail,
			},
			Enabled: currency.Pairs{
				enabled,
			},
		},
	}

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair delimiter and index should throw an error")
	}

	// Test wrong pair delimiter throws an error
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].ConfigFormat.Index = ""
	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair delimiter should throw an error")
	}

	// Test wrong pair index in the enabled pairs throw an error
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		ConfigFormat: &currency.PairFormat{
			Index: currency.AUD.String(),
		},
	}
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPair(currency.BTC, currency.AUD),
	}
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPair(currency.BTC, currency.KRW),
	}

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair index should throw an error")
	}
}

func TestCheckPairConsistency(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.CheckPairConsistency("asdf"); err == nil {
		t.Error("non-existent exchange should return an error")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	// Test nil pair store
	if err := c.CheckPairConsistency(testFakeExchangeName); err == nil {
		t.Error("nil pair store should return an error")
	}

	enabled, err := currency.NewPairDelimiter("BTC_USD", "_")
	if err != nil {
		t.Fatal(err)
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				RequestFormat: &currency.PairFormat{
					Uppercase: false,
					Delimiter: "_",
				},
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
					Delimiter: "_",
				},
				Enabled: currency.Pairs{
					enabled,
				},
			},
		},
	}

	// Test for nil avail pairs
	err = c.CheckPairConsistency(testFakeExchangeName)
	if err != nil {
		t.Error(err)
	}

	p1, err := currency.NewPairDelimiter("LTC_USD", "_")
	if err != nil {
		t.Fatal(err)
	}

	// Test that enabled pair is not found in the available pairs
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		p1,
	}

	// LTC_USD is only found in the available pairs list and should therefor
	// be added to the enabled pairs list due to the atLestOneEnabled code
	err = c.CheckPairConsistency(testFakeExchangeName)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled {
		if !item.Equal(p1) {
			t.Fatal("LTC_USD should be contained in the enabled pairs list")
		}
	}

	p2, err := currency.NewPairDelimiter("BTC_USD", "_")
	if err != nil {
		t.Fatal(err)
	}

	// Add the BTC_USD pair and see result
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		p1,
		p2,
	}

	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Fatal(err)
	}

	// Test that an empty enabled pair is populated with an available pair
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = nil
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	if len(c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled) != 1 {
		t.Fatal("should be populated with atleast one currency pair")
	}

	// Test that an invalid enabled pair is removed from the list
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		p1,
		p2,
	}
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	// Test when no update is required as the available pairs and enabled pairs
	// are consistent
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(true)
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{}

	// Test no conflict and atleast one on enabled asset type
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(true)
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{currency.NewPair(currency.DASH, currency.USD)}

	// Test with conflict and atleast one on enabled asset type
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(false)
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{}

	// Test no conflict and atleast one on disabled asset type
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPair(currency.DASH, currency.USD),
		p1,
		p2,
	}

	// Test with conflict and atleast one on disabled asset type
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].AssetEnabled = nil

	// assetType enabled failure check
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}
}

func TestSupportsPair(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestSupportsPair. LoadConfig Error: %s", err.Error(),
		)
	}

	assetType := asset.Spot
	if cfg.SupportsPair("asdf",
		currency.NewPair(currency.BTC, currency.USD), assetType) {
		t.Error(
			"TestSupportsPair. Expected error from Non-existent exchange",
		)
	}

	if !cfg.SupportsPair("Bitfinex",
		currency.NewPair(currency.BTC, currency.USD), assetType) {
		t.Errorf(
			"TestSupportsPair. Incorrect values. Err: %s", err,
		)
	}
}

func TestGetPairFormat(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetPairFormat("meow", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		UseGlobalFormat: false,
		RequestFormat: &currency.PairFormat{
			Uppercase: false,
			Delimiter: "_",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: nil,
		},
	}

	_, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: false,
			Delimiter: "_",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: new(currency.PairStore),
		},
	}
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Item("invalid"))
	if err == nil {
		t.Error("Expected error from non-existent asset item")
	}

	_, err = c.GetPairFormat(testFakeExchangeName, asset.Futures)
	if err == nil {
		t.Error("Expected error from valid but non supported asset type")
	}

	var p currency.PairFormat
	p, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if !p.Uppercase && p.Delimiter != "_" {
		t.Error("unexpected results")
	}

	// Test nil pair store
	c.Exchanges[0].CurrencyPairs.UseGlobalFormat = false
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "~",
			},
		},
	}
	p, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if p.Delimiter != "~" && !p.Uppercase {
		t.Error("unexpected results")
	}
}

func TestGetAvailablePairs(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetAvailablePairs("asdf", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name:          testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{},
		},
	)

	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
		},
	}
	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error("Expected error from nil pairs")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
	}
	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEnabledPairs(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetEnabledPairs("asdf", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name:          testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{},
		},
	)

	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
		},
	}
	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error("nil pairs should return a nil error")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
	}

	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEnabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestGetEnabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetEnabledExchanges()
	if len(exchanges) != defaultEnabledExchanges {
		t.Error(
			"TestGetEnabledExchanges. Enabled exchanges value mismatch",
		)
	}

	if !common.StringDataCompare(exchanges, "Bitfinex") {
		t.Error(
			"TestGetEnabledExchanges. Expected exchange Bitfinex not found",
		)
	}
}

func TestGetDisabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetDisabledExchanges()
	if len(exchanges) != 0 {
		t.Error(
			"TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. GetExchangeConfig Error: %s", err.Error(),
		)
	}

	exchCfg.Enabled = false
	err = cfg.UpdateExchangeConfig(exchCfg)
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. UpdateExchangeConfig Error: %s", err.Error(),
		)
	}

	if len(cfg.GetDisabledExchanges()) != 1 {
		t.Error(
			"TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}
}

func TestCountEnabledExchanges(t *testing.T) {
	GetConfigEnabledExchanges := GetConfig()
	err := GetConfigEnabledExchanges.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(
			"GetConfigEnabledExchanges load config error: " + err.Error(),
		)
	}
	enabledExch := GetConfigEnabledExchanges.CountEnabledExchanges()
	if enabledExch != defaultEnabledExchanges {
		t.Errorf("Expected %v, Received %v", defaultEnabledExchanges, enabledExch)
	}
}

func TestGetCurrencyPairDisplayConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"GetCurrencyPairDisplayConfig. LoadConfig Error: %s", err.Error(),
		)
	}
	settings := cfg.GetCurrencyPairDisplayConfig()
	if settings.Delimiter != "-" || !settings.Uppercase {
		t.Errorf(
			"GetCurrencyPairDisplayConfi. Invalid values",
		)
	}
}

func TestGetAllExchangeConfigs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetAllExchangeConfigs. LoadConfig error", err)
	}
	if len(cfg.GetAllExchangeConfigs()) < 26 {
		t.Error("GetAllExchangeConfigs error")
	}
}

func TestGetExchangeConfig(t *testing.T) {
	GetExchangeConfig := GetConfig()
	err := GetExchangeConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"GetExchangeConfig.LoadConfig Error: %s", err.Error(),
		)
	}
	_, err = GetExchangeConfig.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf("GetExchangeConfig.GetExchangeConfig Error: %s",
			err.Error())
	}
	_, err = GetExchangeConfig.GetExchangeConfig("Testy")
	if err == nil {
		t.Error("GetExchangeConfig.GetExchangeConfig Expected error")
	}
}

func TestGetForexProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetForexProviderConfig. LoadConfig error", err)
	}
	_, err = cfg.GetForexProvider("Fixer")
	if err != nil {
		t.Error("GetForexProviderConfig error", err)
	}

	_, err = cfg.GetForexProvider("this is not a forex provider")
	if err == nil {
		t.Error("GetForexProviderConfig no error for invalid provider")
	}
}

func TestGetForexProviders(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(err)
	}

	if r := cfg.GetForexProviders(); len(r) != 5 {
		t.Error("unexpected length of forex providers")
	}
}

func TestGetPrimaryForexProvider(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetPrimaryForexProvider. LoadConfig error", err)
	}
	primary := cfg.GetPrimaryForexProvider()
	if primary == "" {
		t.Error("GetPrimaryForexProvider error")
	}

	for i := range cfg.Currency.ForexProviders {
		cfg.Currency.ForexProviders[i].PrimaryProvider = false
	}
	primary = cfg.GetPrimaryForexProvider()
	if primary != "" {
		t.Error("GetPrimaryForexProvider error, expected nil got:", primary)
	}
}

func TestUpdateExchangeConfig(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(err)
	}

	e := &ExchangeConfig{}
	err = c.UpdateExchangeConfig(e)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	e, err = c.GetExchangeConfig("OKEX")
	if err != nil {
		t.Error(err)
	}

	e.API.Credentials.Key = "test1234"
	err = c.UpdateExchangeConfig(e)
	if err != nil {
		t.Error(err)
	}
}

// TestCheckExchangeConfigValues logic test
func TestCheckExchangeConfigValues(t *testing.T) {
	var cfg Config
	if err := cfg.CheckExchangeConfigValues(); err == nil {
		t.Error("nil exchanges should throw an err")
	}

	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Fatal(err)
	}

	// Test our default test config and report any errors
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Exchanges[0].Name = "GDAX"
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}
	if cfg.Exchanges[0].Name != "CoinbasePro" {
		t.Error("exchange name should have been updated from GDAX to CoinbasePRo")
	}

	// Test API settings migration
	sptr := func(s string) *string { return &s }
	int64ptr := func(i int64) *int64 { return &i }

	cfg.Exchanges[0].APIKey = sptr("awesomeKey")
	cfg.Exchanges[0].APISecret = sptr("meowSecret")
	cfg.Exchanges[0].ClientID = sptr("clientIDerino")
	cfg.Exchanges[0].APIAuthPEMKey = sptr("-----BEGIN EC PRIVATE KEY-----\nASDF\n-----END EC PRIVATE KEY-----\n")
	cfg.Exchanges[0].APIAuthPEMKeySupport = convert.BoolPtr(true)
	cfg.Exchanges[0].AuthenticatedAPISupport = convert.BoolPtr(true)
	cfg.Exchanges[0].AuthenticatedWebsocketAPISupport = convert.BoolPtr(true)
	cfg.Exchanges[0].WebsocketURL = sptr("wss://1337")
	cfg.Exchanges[0].APIURL = sptr(APIURLNonDefaultMessage)
	cfg.Exchanges[0].APIURLSecondary = sptr(APIURLNonDefaultMessage)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	// Ensure that all of our previous settings are migrated
	if cfg.Exchanges[0].API.Credentials.Key != "awesomeKey" ||
		cfg.Exchanges[0].API.Credentials.Secret != "meowSecret" ||
		cfg.Exchanges[0].API.Credentials.ClientID != "clientIDerino" ||
		!strings.Contains(cfg.Exchanges[0].API.Credentials.PEMKey, "ASDF") ||
		!cfg.Exchanges[0].API.PEMKeySupport ||
		!cfg.Exchanges[0].API.AuthenticatedSupport ||
		!cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("unexpected values")
	}

	if cfg.Exchanges[0].APIKey != nil ||
		cfg.Exchanges[0].APISecret != nil ||
		cfg.Exchanges[0].ClientID != nil ||
		cfg.Exchanges[0].APIAuthPEMKey != nil ||
		cfg.Exchanges[0].APIAuthPEMKeySupport != nil ||
		cfg.Exchanges[0].AuthenticatedAPISupport != nil ||
		cfg.Exchanges[0].AuthenticatedWebsocketAPISupport != nil ||
		cfg.Exchanges[0].WebsocketURL != nil ||
		cfg.Exchanges[0].APIURL != nil ||
		cfg.Exchanges[0].APIURLSecondary != nil {
		t.Error("unexpected values")
	}

	// Test feature and endpoint migrations migrations
	cfg.Exchanges[0].Features = nil
	cfg.Exchanges[0].SupportsAutoPairUpdates = convert.BoolPtr(true)
	cfg.Exchanges[0].Websocket = convert.BoolPtr(true)

	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if !cfg.Exchanges[0].Features.Enabled.AutoPairUpdates ||
		!cfg.Exchanges[0].Features.Enabled.Websocket ||
		!cfg.Exchanges[0].Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Error("unexpected values")
	}

	p1, err := currency.NewPairDelimiter(testPair, "-")
	if err != nil {
		t.Fatal(err)
	}

	// Test currency pair migration
	setupPairs := func(emptyAssets bool) {
		cfg.Exchanges[0].CurrencyPairs = nil
		p := currency.Pairs{
			p1,
		}
		cfg.Exchanges[0].PairsLastUpdated = int64ptr(1234567)

		if !emptyAssets {
			cfg.Exchanges[0].AssetTypes = sptr("spot")
		}

		cfg.Exchanges[0].AvailablePairs = &p
		cfg.Exchanges[0].EnabledPairs = &p
		cfg.Exchanges[0].ConfigCurrencyPairFormat = &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		}
		cfg.Exchanges[0].RequestCurrencyPairFormat = &currency.PairFormat{
			Uppercase: false,
			Delimiter: "~",
		}
	}

	setupPairs(false)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	setupPairs(true)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if cfg.Exchanges[0].CurrencyPairs.LastUpdated != 1234567 {
		t.Error("last updated has wrong value")
	}

	pFmt := cfg.Exchanges[0].CurrencyPairs.ConfigFormat
	if pFmt.Delimiter != "-" ||
		!pFmt.Uppercase {
		t.Error("unexpected config format values")
	}

	pFmt = cfg.Exchanges[0].CurrencyPairs.RequestFormat
	if pFmt.Delimiter != "~" ||
		pFmt.Uppercase {
		t.Error("unexpected request format values")
	}

	if !cfg.Exchanges[0].CurrencyPairs.GetAssetTypes().Contains(asset.Spot) ||
		!cfg.Exchanges[0].CurrencyPairs.UseGlobalFormat {
		t.Error("unexpected results")
	}

	pairs, err := cfg.Exchanges[0].CurrencyPairs.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(pairs) == 0 || pairs.Join() != testPair {
		t.Error("pairs not set properly")
	}

	pairs, err = cfg.Exchanges[0].CurrencyPairs.GetPairs(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(pairs) == 0 || pairs.Join() != testPair {
		t.Error("pairs not set properly")
	}

	// Ensure that all old settings are flushed
	if cfg.Exchanges[0].PairsLastUpdated != nil ||
		cfg.Exchanges[0].ConfigCurrencyPairFormat != nil ||
		cfg.Exchanges[0].RequestCurrencyPairFormat != nil ||
		cfg.Exchanges[0].AssetTypes != nil ||
		cfg.Exchanges[0].AvailablePairs != nil ||
		cfg.Exchanges[0].EnabledPairs != nil {
		t.Error("unexpected results")
	}

	// Test AutoPairUpdates
	cfg.Exchanges[0].Features.Supports.RESTCapabilities.AutoPairUpdates = false
	cfg.Exchanges[0].Features.Supports.WebsocketCapabilities.AutoPairUpdates = false
	cfg.Exchanges[0].CurrencyPairs.LastUpdated = 0
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	// Test websocket and HTTP timeout values
	cfg.Exchanges[0].WebsocketResponseMaxLimit = 0
	cfg.Exchanges[0].WebsocketResponseCheckTimeout = 0
	cfg.Exchanges[0].OrderbookConfig.WebsocketBufferLimit = 0
	cfg.Exchanges[0].WebsocketTrafficTimeout = 0
	cfg.Exchanges[0].HTTPTimeout = 0
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if cfg.Exchanges[0].WebsocketResponseMaxLimit == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketResponseMaxLimit value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].OrderbookConfig.WebsocketBufferLimit == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketOrderbookBufferLimit value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].WebsocketTrafficTimeout == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketTrafficTimeout value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].HTTPTimeout == 0 {
		t.Errorf("expected exchange %s to have updated HTTPTimeout value",
			cfg.Exchanges[0].Name)
	}

	v := &APICredentialsValidatorConfig{
		RequiresKey:    true,
		RequiresSecret: true,
	}
	cfg.Exchanges[0].API.CredentialsValidator = v
	cfg.Exchanges[0].API.Credentials.Key = "Key"
	cfg.Exchanges[0].API.Credentials.Secret = "Secret"
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].API.AuthenticatedSupport ||
		cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected authenticated endpoints to be false from invalid API keys")
	}

	v.RequiresKey = false
	v.RequiresClientID = true
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.Exchanges[0].API.Credentials.ClientID = DefaultAPIClientID
	cfg.Exchanges[0].API.Credentials.Secret = "TESTYTEST"
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].API.AuthenticatedSupport ||
		cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected AuthenticatedAPISupport to be false from invalid API keys")
	}

	v.RequiresKey = true
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.Exchanges[0].API.Credentials.Key = "meow"
	cfg.Exchanges[0].API.Credentials.Secret = "test123"
	cfg.Exchanges[0].API.Credentials.ClientID = "clientIDerino"
	cfg.CheckExchangeConfigValues()
	if !cfg.Exchanges[0].API.AuthenticatedSupport ||
		!cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected AuthenticatedAPISupport and AuthenticatedWebsocketAPISupport to be false from invalid API keys")
	}

	// Make a sneaky copy for bank account testing
	cpy := append(cfg.Exchanges[:0:0], cfg.Exchanges...)

	// Test empty exchange name for an enabled exchange
	cfg.Exchanges[0].Enabled = true
	cfg.Exchanges[0].Name = ""
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].Enabled {
		t.Errorf(
			"Exchange with no name should be empty",
		)
	}

	// Test no enabled exchanges
	cfg.Exchanges = cfg.Exchanges[:1]
	cfg.Exchanges[0].Enabled = false
	err = cfg.CheckExchangeConfigValues()
	if err == nil {
		t.Error("Expected error from no enabled exchanges")
	}

	cfg.Exchanges = cpy
	// Check bank account validation for exchange
	cfg.Exchanges[0].BankAccounts = []banking.Account{
		{
			Enabled: true,
		},
	}

	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if cfg.Exchanges[0].BankAccounts[0].Enabled {
		t.Fatal("bank aaccount details not provided this should disable")
	}

	// Test international bank
	cfg.Exchanges[0].BankAccounts[0].Enabled = true
	cfg.Exchanges[0].BankAccounts[0].BankName = testString
	cfg.Exchanges[0].BankAccounts[0].BankAddress = testString
	cfg.Exchanges[0].BankAccounts[0].BankPostalCode = testString
	cfg.Exchanges[0].BankAccounts[0].BankPostalCity = testString
	cfg.Exchanges[0].BankAccounts[0].BankCountry = testString
	cfg.Exchanges[0].BankAccounts[0].AccountName = testString
	cfg.Exchanges[0].BankAccounts[0].SupportedCurrencies = "monopoly moneys"
	cfg.Exchanges[0].BankAccounts[0].IBAN = "some iban"
	cfg.Exchanges[0].BankAccounts[0].SWIFTCode = "some swifty"

	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if !cfg.Exchanges[0].BankAccounts[0].Enabled {
		t.Fatal("bank aaccount details provided this should not disable")
	}

	// Test aussie bank
	cfg.Exchanges[0].BankAccounts[0].Enabled = true
	cfg.Exchanges[0].BankAccounts[0].BankName = testString
	cfg.Exchanges[0].BankAccounts[0].BankAddress = testString
	cfg.Exchanges[0].BankAccounts[0].BankPostalCode = testString
	cfg.Exchanges[0].BankAccounts[0].BankPostalCity = testString
	cfg.Exchanges[0].BankAccounts[0].BankCountry = testString
	cfg.Exchanges[0].BankAccounts[0].AccountName = testString
	cfg.Exchanges[0].BankAccounts[0].SupportedCurrencies = "AUD"
	cfg.Exchanges[0].BankAccounts[0].BSBNumber = "some BSB nonsense"
	cfg.Exchanges[0].BankAccounts[0].IBAN = ""
	cfg.Exchanges[0].BankAccounts[0].SWIFTCode = ""

	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if !cfg.Exchanges[0].BankAccounts[0].Enabled {
		t.Fatal("bank account details provided this should not disable")
	}

	cfg.Exchanges = nil
	cfg.Exchanges = append(cfg.Exchanges, cpy[0])

	cfg.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = nil
	cfg.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(false)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	cfg.Exchanges[0].CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	err = cfg.CheckExchangeConfigValues()
	if err == nil {
		t.Error("err cannot be nil")
	}
}

func TestRetrieveConfigCurrencyPairs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.LoadConfig: %s", err.Error(),
		)
	}
	err = cfg.RetrieveConfigCurrencyPairs(true, asset.Spot)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}

	err = cfg.RetrieveConfigCurrencyPairs(false, asset.Spot)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}
}

func TestReadConfigFromFile(t *testing.T) {
	readConfig := GetConfig()
	err := readConfig.ReadConfigFromFile(TestFile, true)
	if err != nil {
		t.Errorf("TestReadConfig %s", err.Error())
	}

	err = readConfig.ReadConfigFromFile("bla", true)
	if err == nil {
		t.Error("TestReadConfig error cannot be nil")
	}
}

func TestReadConfigFromReader(t *testing.T) {
	confString := `{"name":"test"}`
	conf, encrypted, err := ReadConfig(strings.NewReader(confString), Unencrypted)
	if err != nil {
		t.Errorf("TestReadConfig %s", err)
	}
	if encrypted {
		t.Errorf("Expected unencrypted config %s", err)
	}
	if conf.Name != "test" {
		t.Errorf("Conf not properly loaded %s", err)
	}

	_, _, err = ReadConfig(strings.NewReader("{}"), Unencrypted)
	if err == nil {
		t.Error("TestReadConfig error cannot be nil")
	}
}

func TestLoadConfig(t *testing.T) {
	loadConfig := GetConfig()
	err := loadConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("TestLoadConfig " + err.Error())
	}

	err = loadConfig.LoadConfig("testy", true)
	if err == nil {
		t.Error("TestLoadConfig Expected error")
	}
}

func TestSaveConfigToFile(t *testing.T) {
	saveConfig := GetConfig()
	err := saveConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("TestSaveConfig.LoadConfig: %s", err.Error())
	}
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("TestSaveConfig create file: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())
	err2 := saveConfig.SaveConfigToFile(f.Name())
	if err2 != nil {
		t.Errorf("TestSaveConfig.SaveConfig, %s", err2.Error())
	}
}

func TestCheckConnectionMonitorConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.ConnectionMonitor.CheckInterval = 0
	c.ConnectionMonitor.DNSList = nil
	c.ConnectionMonitor.PublicDomainList = nil
	c.CheckConnectionMonitorConfig()

	if c.ConnectionMonitor.CheckInterval != connchecker.DefaultCheckInterval ||
		len(common.StringSliceDifference(
			c.ConnectionMonitor.DNSList, connchecker.DefaultDNSList)) != 0 ||
		len(common.StringSliceDifference(
			c.ConnectionMonitor.PublicDomainList, connchecker.DefaultDomainList)) != 0 {
		t.Error("unexpected values")
	}
}

func TestDefaultFilePath(t *testing.T) {
	// This is tricky to test because we're dealing with a config file stored
	// in a persons default directory and to properly test it, it would
	// require causing os.Stat to return !os.IsNotExist and os.IsNotExist (which
	// means moving a users config file around), a way of getting around this is
	// to pass the datadir as a param line but adds a burden to everyone who
	// uses it
	result := DefaultFilePath()
	if !strings.Contains(result, File) &&
		!strings.Contains(result, EncryptedFile) {
		t.Error("result should have contained config.json or config.dat")
	}
}

func TestGetFilePath(t *testing.T) {
	expected := "blah.json"
	result, wasDefault, _ := GetFilePath("blah.json")
	if result != "blah.json" {
		t.Errorf("TestGetFilePath: expected %s got %s", expected, result)
	}
	if wasDefault {
		t.Errorf("TestGetFilePath: expected non-default")
	}

	expected = DefaultFilePath()
	result, wasDefault, err := GetFilePath("")
	if file.Exists(expected) {
		if err != nil || result != expected {
			t.Errorf("TestGetFilePath: expected %s got %s", expected, result)
		}
		if !wasDefault {
			t.Errorf("TestGetFilePath: expected default file")
		}
	} else if err == nil {
		t.Error("Expected error when default config file does not exist")
	}
}

func TestCheckRemoteControlConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.Webserver = &WebserverConfig{
		Enabled:                      true,
		AdminUsername:                "satoshi",
		AdminPassword:                "ultrasecurepassword",
		ListenAddress:                ":9050",
		WebsocketConnectionLimit:     5,
		WebsocketMaxAuthFailures:     10,
		WebsocketAllowInsecureOrigin: true,
	}

	c.CheckRemoteControlConfig()

	if c.RemoteControl.Username != "satoshi" ||
		c.RemoteControl.Password != "ultrasecurepassword" ||
		!c.RemoteControl.GRPC.Enabled ||
		c.RemoteControl.GRPC.ListenAddress != "localhost:9052" ||
		!c.RemoteControl.GRPC.GRPCProxyEnabled ||
		c.RemoteControl.GRPC.GRPCProxyListenAddress != "localhost:9053" ||
		!c.RemoteControl.DeprecatedRPC.Enabled ||
		c.RemoteControl.DeprecatedRPC.ListenAddress != "localhost:9050" ||
		!c.RemoteControl.WebsocketRPC.Enabled ||
		c.RemoteControl.WebsocketRPC.ListenAddress != "localhost:9051" ||
		!c.RemoteControl.WebsocketRPC.AllowInsecureOrigin ||
		c.RemoteControl.WebsocketRPC.ConnectionLimit != 5 ||
		c.RemoteControl.WebsocketRPC.MaxAuthFailures != 10 {
		t.Error("unexpected results")
	}

	// Now test to ensure the previous settings are flushed
	if c.Webserver != nil {
		t.Error("old webserver settings should be nil")
	}
}

func TestCheckConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = c.CheckConfig()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("%s", err)
	}

	newCfg := c
	err = c.UpdateConfig(TestFile, &newCfg, true)
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = c.UpdateConfig("//non-existantpath\\", &newCfg, false)
	if err == nil {
		t.Fatalf("Error should have been thrown for invalid path")
	}

	newCfg.Currency.Cryptocurrencies = currency.NewCurrenciesFromStringArray([]string{""})
	err = c.UpdateConfig(TestFile, &newCfg, true)
	if err != nil {
		t.Errorf("%s", err)
	}
	if c.Currency.Cryptocurrencies.Join() == "" {
		t.Fatalf("Cryptocurrencies should have been repopulated")
	}
}

func BenchmarkUpdateConfig(b *testing.B) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		b.Errorf("Unable to benchmark UpdateConfig(): %s", err)
	}

	newCfg := c
	for i := 0; i < b.N; i++ {
		_ = c.UpdateConfig(TestFile, &newCfg, true)
	}
}

func TestCheckLoggerConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.Logging = log.Config{}
	err := c.CheckLoggerConfig()
	if err != nil {
		t.Errorf("Failed to create default logger. Error: %s", err)
	}

	if !*c.Logging.Enabled {
		t.Error("unexpected result")
	}

	c.Logging.LoggerFileConfig.FileName = ""
	c.Logging.LoggerFileConfig.Rotate = nil
	c.Logging.LoggerFileConfig.MaxSize = -1
	c.Logging.AdvancedSettings.ShowLogSystemName = nil

	err = c.CheckLoggerConfig()
	if err != nil {
		t.Error(err)
	}

	if c.Logging.LoggerFileConfig.FileName != "log.txt" ||
		c.Logging.LoggerFileConfig.Rotate == nil ||
		c.Logging.LoggerFileConfig.MaxSize != 100 ||
		c.Logging.AdvancedSettings.ShowLogSystemName == nil ||
		*c.Logging.AdvancedSettings.ShowLogSystemName {
		t.Error("unexpected result")
	}
}

func TestDisableNTPCheck(t *testing.T) {
	t.Parallel()

	var c Config

	warn, err := c.DisableNTPCheck(strings.NewReader("w\n"))
	if err != nil {
		t.Fatalf("to create ntpclient failed reason: %v", err)
	}

	if warn != "Time sync has been set to warn only" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to warn only", warn)
	}
	alert, _ := c.DisableNTPCheck(strings.NewReader("a\n"))
	if alert != "Time sync has been set to alert" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to alert", alert)
	}

	disable, _ := c.DisableNTPCheck(strings.NewReader("d\n"))
	if disable != "Future notifications for out of time sync has been disabled" {
		t.Errorf("failed expected %v got %v", "Future notifications for out of time sync has been disabled", disable)
	}

	_, err = c.DisableNTPCheck(strings.NewReader(" "))
	if err.Error() != "EOF" {
		t.Errorf("failed expected EOF got: %v", err)
	}
}

func TestCheckGCTScriptConfig(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.checkGCTScriptConfig(); err != nil {
		t.Error(err)
	}

	if c.GCTScript.ScriptTimeout != gctscript.DefaultTimeoutValue {
		t.Fatal("unexpected value return")
	}

	if c.GCTScript.MaxVirtualMachines != gctscript.DefaultMaxVirtualMachines {
		t.Fatal("unexpected value return")
	}
}

func TestCheckDatabaseConfig(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.checkDatabaseConfig(); err != nil {
		t.Error(err)
	}

	if c.Database.Driver != database.DBSQLite3 ||
		c.Database.Database != database.DefaultSQLiteDatabase ||
		c.Database.Enabled {
		t.Error("unexpected results")
	}

	c.Database.Enabled = true
	c.Database.Driver = "mssqlisthebest"
	if err := c.checkDatabaseConfig(); err == nil {
		t.Error("unexpected result")
	}

	c.Database.Driver = database.DBSQLite3
	c.Database.Enabled = true
	if err := c.checkDatabaseConfig(); err != nil {
		t.Error(err)
	}
}

func TestCheckNTPConfig(t *testing.T) {
	c := GetConfig()

	c.NTPClient.Level = 0
	c.NTPClient.Pool = nil
	c.NTPClient.AllowedNegativeDifference = nil
	c.NTPClient.AllowedDifference = nil

	c.CheckNTPConfig()
	_ = ntpclient.NTPClient(c.NTPClient.Pool)

	if c.NTPClient.Pool[0] != "pool.ntp.org:123" {
		t.Error("ntpclient with no valid pool should default to pool.ntp.org")
	}

	if c.NTPClient.AllowedDifference == nil {
		t.Error("ntpclient with nil alloweddifference should default to sane value")
	}

	if c.NTPClient.AllowedNegativeDifference == nil {
		t.Error("ntpclient with nil allowednegativedifference should default to sane value")
	}
}

func TestCheckCurrencyConfigValues(t *testing.T) {
	c := GetConfig()
	c.Currency.ForexProviders = nil
	c.Currency.CryptocurrencyProvider = CryptocurrencyProvider{}
	err := c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.Currency.ForexProviders == nil {
		t.Error("Failed to populate c.Currency.ForexProviders")
	}
	if c.Currency.CryptocurrencyProvider.APIkey != DefaultUnsetAPIKey {
		t.Error("Failed to set the api key to the default key")
	}
	if c.Currency.CryptocurrencyProvider.Name != "CoinMarketCap" {
		t.Error("Failed to set the  c.Currency.CryptocurrencyProvider.Name")
	}

	c.Currency.ForexProviders[0].Enabled = true
	c.Currency.ForexProviders[0].Name = "CurrencyConverter"
	c.Currency.ForexProviders[0].PrimaryProvider = true
	c.Currency.Cryptocurrencies = nil
	c.Cryptocurrencies = nil
	c.Currency.CurrencyPairFormat = nil
	c.CurrencyPairFormat = &CurrencyPairFormatConfig{
		Uppercase: true,
	}
	c.Currency.FiatDisplayCurrency = currency.Code{}
	c.FiatDisplayCurrency = &currency.BTC
	c.Currency.CryptocurrencyProvider.Enabled = true
	err = c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.Currency.ForexProviders[0].Enabled {
		t.Error("Failed to disable invalid forex provider")
	}
	if !c.Currency.CurrencyPairFormat.Uppercase {
		t.Error("Failed to apply c.CurrencyPairFormat format to c.Currency.CurrencyPairFormat")
	}

	c.Currency.CryptocurrencyProvider.Enabled = false
	c.Currency.CryptocurrencyProvider.APIkey = ""
	c.Currency.CryptocurrencyProvider.AccountPlan = ""
	c.FiatDisplayCurrency = &currency.BTC
	c.Currency.ForexProviders[0].Enabled = true
	c.Currency.ForexProviders[0].Name = "Name"
	c.Currency.ForexProviders[0].PrimaryProvider = true
	c.Currency.Cryptocurrencies = currency.Currencies{}
	c.Cryptocurrencies = &currency.Currencies{}
	err = c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.FiatDisplayCurrency != nil {
		t.Error("Failed to clear c.FiatDisplayCurrency")
	}
	if c.Currency.CryptocurrencyProvider.APIkey != DefaultUnsetAPIKey ||
		c.Currency.CryptocurrencyProvider.AccountPlan != DefaultUnsetAccountPlan {
		t.Error("Failed to set CryptocurrencyProvider.APIkey and AccountPlan")
	}
}

func TestPreengineConfigUpgrade(t *testing.T) {
	var c Config
	if err := c.LoadConfig("../testdata/preengine_config.json", false); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveExchange(t *testing.T) {
	t.Parallel()
	var c Config
	const testExchangeName = "0xBAAAAAAD"
	c.Exchanges = append(c.Exchanges, ExchangeConfig{
		Name: testExchangeName,
	})
	_, err := c.GetExchangeConfig(testExchangeName)
	if err != nil {
		t.Fatal(err)
	}
	if success := c.RemoveExchange(testExchangeName); !success {
		t.Fatal("exchange should of been removed")
	}
	_, err = c.GetExchangeConfig(testExchangeName)
	if err == nil {
		t.Fatal("non-existent exchange should throw an error")
	}
	if success := c.RemoveExchange("1D10TH0RS3"); success {
		t.Fatal("exchange shouldn't exist")
	}
}

func TestGetDataPath(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		elem []string
		want string
	}{
		{
			name: "empty",
			dir:  "",
			elem: []string{},
			want: common.GetDefaultDataDir(runtime.GOOS),
		},
		{
			name: "empty a b",
			dir:  "",
			elem: []string{"a", "b"},
			want: filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "a", "b"),
		},
		{
			name: "target",
			dir:  "target",
			elem: []string{"a", "b"},
			want: filepath.Join("target", "a", "b"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{
				DataDirectory: tt.dir,
			}
			if got := c.GetDataPath(tt.elem...); got != tt.want {
				t.Errorf("Config.GetDataPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrateConfig(t *testing.T) {
	type args struct {
		configFile string
		targetDir  string
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tests := []struct {
		name    string
		setup   func(t *testing.T)
		cleanup func(t *testing.T)
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "nonexisting",
			args: args{
				configFile: "not-exists.json",
			},
			wantErr: true,
		},
		{
			name: "source present, no target dir",
			setup: func(t *testing.T) {
				test, err := os.Create("test.json")
				if err != nil {
					t.Fatal(err)
				}
				test.Close()
			},
			cleanup: func(t *testing.T) {
				os.Remove("test.json")
			},
			args: args{
				configFile: "test.json",
				targetDir:  filepath.Join(dir, "new"),
			},
			want:    filepath.Join(dir, "new", File),
			wantErr: false,
		},
		{
			name: "source same as target",
			setup: func(t *testing.T) {
				err := file.Write(filepath.Join(dir, File), nil)
				if err != nil {
					t.Fatal(err)
				}
			},
			args: args{
				configFile: filepath.Join(dir, File),
				targetDir:  dir,
			},
			want:    filepath.Join(dir, File),
			wantErr: false,
		},
		{
			name: "source and target present",
			setup: func(t *testing.T) {
				err := file.Write(filepath.Join(dir, File), nil)
				if err != nil {
					t.Fatal(err)
				}
				err = file.Write(filepath.Join(dir, "src", EncryptedFile), nil)
				if err != nil {
					t.Fatal(err)
				}
			},
			args: args{
				configFile: filepath.Join(dir, "src", EncryptedFile),
				targetDir:  dir,
			},
			want: filepath.Join(dir, "src", EncryptedFile),
			// We only expect warning
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			if tt.cleanup != nil {
				defer tt.cleanup(t)
			}
			got, err := migrateConfig(tt.args.configFile, tt.args.targetDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("migrateConfig() = %v, want %v", got, tt.want)
			}
			if err == nil && !file.Exists(got) {
				t.Errorf("migrateConfig: %v should exist", got)
			}
		})
	}
}

func TestEnsureTestConfigHasNotBeenModified(t *testing.T) {
	configFileHash := "d5dc7c7d8a0074451aab33154dd74beceabe4f5e0e67139ea78423ce3173d09f"
	p := filepath.Join("..", "testdata", "configtest.json")

	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	currentConfigHash := fmt.Sprintf("%x", h.Sum(nil))
	if currentConfigHash != configFileHash {
		t.Errorf("please check that the configtest.json file has not been updated. If a necessary change has been made, please update the checksum %v %v",
			currentConfigHash,
			configFileHash)
	}
}
