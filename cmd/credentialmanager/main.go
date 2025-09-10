package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account/credentialstore"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// dir is the default file path for credentials
	dir string
	// isEncrypted is a flag to determine if the config file is encrypted
	isEncrypted bool
	// in-memory config, loaded from file and ready to be saved
	cfg *credentialstore.Config
	// the key used to decrypt the config file
	cryptKey string
)

// init function sets the default file path for credentials
func init() {
	dir = credentialstore.DefaultFilePath()
}

func main() {
	fmt.Println("Checking for existing credentials file at " + dir)
	var err error
	cfg, err = credentialstore.SetupConfig(dir, false)
	if err != nil {
		if errors.Is(err, credentialstore.ErrNoCredentialsFile) {
			fmt.Println("No credentials file found. Want to create one? (y/n)")
			choice := readLine()
			if strings.ToLower(choice) == "y" {
				cfg, err = credentialstore.SetupConfig(dir, false)
				if err != nil {
					fmt.Println("Failed to create credentials file:", err)
					fmt.Println("Exiting...")
					os.Exit(1)
				}
			} else {
				fmt.Println("Exiting...")
				os.Exit(0)
			}
		}
	}
	if cfg.IsEncrypted {
		isEncrypted = true
	}
	defaultMenu()
}

// defaultMenu displays the main menu and handles user choices
func defaultMenu() {
	fmt.Println("What do you want to do? (Enter the number)")
	fmt.Println("Please note: changes are only saved when en/decrypting, or on save & exit")
	fmt.Println("1. Add new credentials")
	fmt.Println("2. Disable existing credentials")
	fmt.Println("3. Enable inactive credentials")
	fmt.Println("4. Delete existing credentials")
	fmt.Println("5. Delete all credentials")
	fmt.Println("6. List all credentials")
	if isEncrypted {
		fmt.Println("7. Decrypt config file at rest (Also saves)")
	} else {
		fmt.Println("7. Encrypt config file at rest (Also saves)")
	}
	fmt.Println("8. Save & Exit")
	fmt.Println("9. Exit without saving")
	updateChoice := readLine()
	switch updateChoice {
	case "1":
		addNewCredentials()
	case "2":
		disableExistingCredentials()
	case "3":
		enableExistingCredentials()
	case "4":
		deleteCredentials()
	case "5":
		deleteAllCredentials()
	case "6":
		listAllCredentialsSummary()
	case "7":
		if isEncrypted {
			jsonCfg, err := json.MarshalIndent(cfg, "", " ")
			if err != nil {
				panic(err)
			}
			err = os.WriteFile(dir, jsonCfg, 0777)
			if err != nil {
				panic(err)
			}
			isEncrypted = false
		} else {
			fmt.Println("What is the new encryption key?")
			cryptKey = readLine()
			jsonCfg, err := json.MarshalIndent(cfg, "", " ")
			if err != nil {
				panic(err)
			}
			encrypted, err := config.EncryptConfigData(jsonCfg, []byte(cryptKey))
			err = os.WriteFile(dir, encrypted, 0777)
			if err != nil {
				panic(err)
			}
			isEncrypted = true
		}
	case "8":
		fmt.Println("Saving and exiting...")
		jsoncfg, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			panic(err)
		}
		if isEncrypted {
			encrypted, err := config.EncryptConfigData(jsoncfg, []byte(cryptKey))
			if err != nil {
				panic(err)
			}
			err = os.WriteFile(dir, encrypted, 0777)
			if err != nil {
				panic(err)
			}
		} else {
			err = os.WriteFile(dir, jsoncfg, 0777)
			if err != nil {
				panic(err)
			}
		}
		os.Exit(0)
	case "9":
		fmt.Println("Disregarding your changes and exiting...")
		os.Exit(0)
	default:
		fmt.Println("Invalid choice. Returning to main menu.")
	}
	defaultMenu()
}

// disableExistingCredentials allows the user to disable a specific credential
func disableExistingCredentials() {
	fmt.Println("What credentials do you want to disable?")
	listAllCredentialsSummary()
	fmt.Println("Enter the index of the credential you want to disable (eg '7'):")
	var index string
	_, err := fmt.Scanln(&index)
	if err != nil {
		fmt.Println("Invalid input. Returning to main menu.")
		return
	}
	i, err := strconv.Atoi(index)
	if err != nil {
		fmt.Println("Invalid input. Returning to main menu.")
		return
	}
	i-- // well because we start at 1 to be friendly
	cfg.ExchangeAssetPairCredentials[i].Enabled = false
	fmt.Println("Credential disabled successfully.")
}

// enableExistingCredentials allows the user to enable a specific credential
func enableExistingCredentials() {
	fmt.Println("What credentials do you want to enable?")
	listAllCredentialsSummary()
	fmt.Println("Enter the index of the credential you want to enable (eg '1-3'):")
	var index string
	_, err := fmt.Scanln(&index)
	if err != nil {
		fmt.Println("Invalid input. Returning to main menu.")
		return
	}
	i, err := strconv.Atoi(index)
	if err != nil {
		fmt.Println("Invalid input. Returning to main menu.")
		return
	}
	i-- // well because we start at 1 to be friendly
	cfg.ExchangeAssetPairCredentials[i].Enabled = true
	fmt.Println("Credential enabled successfully.")
}

// addNewCredentials prompts the user to input new credential information
func addNewCredentials() {
	var a asset.Item
	var cp currency.Pair
	var err error
	fmt.Println("What exchange are you adding credentials for?")
	exch := readLine()
	fmt.Println("Do you want them to apply to a specific asset? (y/n)")
	choiceText := readLine()
	if choiceText == "y" || choiceText == "Y" {
		fmt.Println("What asset are you adding credentials for?")
		as := readLine()
		a, err = asset.New(as)
		if err != nil {
			panic(err)
		}
		fmt.Println("Do you want them to apply to a specific currency pair? (y/n)")
		choiceText = readLine()
		if choiceText == "y" || choiceText == "Y" {
			fmt.Println("What currency pair are you adding credentials for? (eg 'BTC-USD')")
			pair := readLine()
			cp, err = currency.NewPairFromString(pair)
			if err != nil {
				panic(err)
			}
			cp = cp.Upper()
		}
	}

	fmt.Println("What is the key?")
	apiKey := readLine()
	fmt.Println("What is the secret?")
	apiSecret := readLine()
	fmt.Println("What is the clientID?")
	clientID := readLine()
	fmt.Println("Is the key and secret base64 encoded? (y/n)")
	choiceText = readLine()
	var base64Encoded bool
	if choiceText == "y" || choiceText == "Y" {
		base64Encoded = true
	}
	fmt.Println("What is the subaccount? (leave blank if not applicable)")
	subAccount := readLine()
	c := &account.Credentials{
		Key:                 apiKey,
		Secret:              apiSecret,
		ClientID:            clientID,
		SubAccount:          subAccount,
		SecretBase64Decoded: base64Encoded,
	}
	if cp.IsEmpty() {
		cfg.ExchangeAssetPairCredentials = append(cfg.ExchangeAssetPairCredentials,
			credentialstore.ExchangeAssetPairCredentials{
				Key: &key.ExchangeAssetPair{
					Exchange: exch,
					Asset:    a,
				},
				Credentials: c,
				Enabled:     true,
			})
	} else {
		cfg.ExchangeAssetPairCredentials = append(cfg.ExchangeAssetPairCredentials,
			credentialstore.ExchangeAssetPairCredentials{
				Key: &key.ExchangeAssetPair{
					Exchange: exch,
					Asset:    a,
					Base:     cp.Base.Item,
					Quote:    cp.Quote.Item,
				},
				Credentials: c,
				Enabled:     true,
			})
	}
}

// deleteCredentials allows the user to delete a specific credential
func deleteCredentials() {
	fmt.Println("What credentials do you want to delete?")
	listAllCredentialsSummary()
	fmt.Println("Enter the index of the credential you want to delete (eg '7'):")
	index := readLine()
	i, err := strconv.Atoi(index)
	if err != nil {
		fmt.Println("Invalid input. Returning to main menu.")
		return
	}
	if len(cfg.ExchangeAssetPairCredentials) == 1 && i == 1 {
		cfg.ExchangeAssetPairCredentials = nil
	} else {
		if i == len(cfg.ExchangeAssetPairCredentials) {
			cfg.ExchangeAssetPairCredentials = cfg.ExchangeAssetPairCredentials[:i-1]
		} else if i < len(cfg.ExchangeAssetPairCredentials) {
			i-- // well because we start at 1 to be friendly
			cfg.ExchangeAssetPairCredentials = slices.Delete(cfg.ExchangeAssetPairCredentials, i-1, i+1)
		} else {
			fmt.Println("Invalid input. Returning to main menu.")
			return
		}
	}
	fmt.Println("Credential deleted successfully.")
}

// deleteAllCredentials removes all stored credentials after confirmation
func deleteAllCredentials() {
	fmt.Println("Are you sure you want to delete all credentials? (y/n)")
	confirm := readLine()
	if confirm == "y" || confirm == "Y" {
		cfg.ExchangeAssetPairCredentials = nil
		fmt.Println("All credentials have been deleted.")
	} else {
		fmt.Println("Operation cancelled. Returning to main menu.")
	}
}

// listAllCredentialsSummary displays a summary of all stored credentials
func listAllCredentialsSummary() {
	fmt.Println("Listing all credentials:")
	i := 1
	for _, cred := range cfg.ExchangeAssetPairCredentials {
		if cred.Key.Base != nil && cred.Key.Quote != nil {
			fmt.Printf("%v\t Exch: '%v' Asset: '%v' Pair: '%v' | Enabled: '%v' | Cred: '%v'\n",
				i, cred.Key.Exchange, cred.Key.Asset.String(), cred.Key.Base.String()+"-"+cred.Key.Quote.String(), cred.Enabled, cred.Credentials.String())
		} else {
			fmt.Printf("%v\t Exch: '%v' Asset: '%v' | Enabled: '%v' | Cred: '%v'\n",
				i, cred.Key.Exchange, cred.Key.Asset.String(), cred.Enabled, cred.Credentials.String())
		}
		i++
	}
}

// readLine is a helper function to read a line of input from the user
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
