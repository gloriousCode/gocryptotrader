package credentialstore

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
)

// ErrNoCredentialsFile is returned when no credentials file is found
var ErrNoCredentialsFile = errors.New("no credentials file found, please use the app to make one")

// ExchangeAssetPairCredentials extends the base ExchangeAssetPairCredentials
// with an Enabled field
type ExchangeAssetPairCredentials struct {
	Key         *key.ExchangeAssetPair
	Credentials *account.Credentials
	Enabled     bool `json:"enabled"`
}

// Config represents the structure of the credentials configuration
type Config struct {
	IsEncrypted                  bool                           `json:"-"`
	ExchangeAssetPairCredentials []ExchangeAssetPairCredentials `json:"exchangeAssetPairCredentials"`
}

// SetupConfig initializes the configuration by checking for an existing file
// at the specified path. If the file exists, it attempts to load and parse it.
// If the file does not exist and createFileIfNotExist is true, it creates a new
// empty file at that path. If the file does not exist and createFileIfNotExist
// is false, it returns error.
func SetupConfig(path string, createFileIfNotExist bool) (*Config, error) {
	resp := &Config{}
	if file.Exists(path) {
		// Handle existing credentials file
		fmt.Println("Existing credentials found.")
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		resp, err = SetCredentialsFromFileData(contentBytes)
		if err != nil {
			// assume its encrypted
			fmt.Println("Credentials file is encrypted, please decrypt it first.")
			fmt.Println("What is the encryption key?")
			cryptKey := readLine()
			contentBytes, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			contentBytes, err = config.DecryptConfigData(contentBytes, []byte(cryptKey))
			if err != nil {
				return nil, err
			}
			resp, err = SetCredentialsFromFileData(contentBytes)
			if err != nil {
				fmt.Println("Failed to decrypt config file. Make sure its correct, or start over")
				return nil, err
			}
			resp.IsEncrypted = true
		}
	} else if createFileIfNotExist {
		_, err := os.Create(path)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrNoCredentialsFile
	}
	return resp, nil
}

// SetCredentialsFromFileData unmarshals JSON data into a Config struct
func SetCredentialsFromFileData(data []byte) (*Config, error) {
	var resp Config
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// LoadCredentialsFromPath loads and processes credentials from a file
func LoadCredentialsFromPath(path string) error {
	// Check if the file exists
	if !file.Exists(path) {
		return fmt.Errorf("err: '%s' %w", path, ErrNoCredentialsFile)
	}
	fmt.Println("Existing credentials found.")

	// Read the file contents
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Attempt to parse the credentials
	credConf, err := SetCredentialsFromFileData(contentBytes)
	if err != nil {
		// If parsing fails, assume the file is encrypted
		fmt.Println("Credentials file is encrypted, please decrypt it first.")
		fmt.Println("What is the encryption key?")
		cryptKey := readLine()

		// Decrypt the file contents
		contentBytes, err = config.DecryptConfigData(contentBytes, []byte(cryptKey))
		if err != nil {
			return err
		}

		// Parse the decrypted contents
		credConf, err = SetCredentialsFromFileData(contentBytes)
		if err != nil {
			return err
		}
	}

	return UpsertCredentialsFromConfig(credConf)
}

// readLine is a helper function to read a line of input from the user
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
