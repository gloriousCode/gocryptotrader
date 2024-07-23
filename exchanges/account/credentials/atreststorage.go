package credentials

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type AtRestCredStore struct {
	ExchangePairAssetCredentials []*Credentials
	ExchangeAssetCredentials     []*Credentials
	ExchangeCredentials          []*Credentials
}

type AtRestConfig struct {
	AtRestCredStore AtRestCredStore
	StoredSalt      []byte
	StoredDK        []byte
}

var (
	maxAuthFailures = 3
)

func Load(atRestPath string) error {
	if !file.Exists(atRestPath) {
		return fmt.Errorf("%w %v", common.ErrFileNotFound, atRestPath)
	}
	confFile, err := os.Open(atRestPath)
	if err != nil {
		return err
	}
	defer func() {
		err = confFile.Close()
		if err != nil {
			log.Errorln(log.ConfigMgr, err)
		}
	}()
	reader := bufio.NewReader(configReader)

	pref, err := reader.Peek(len(crypto.EncryptConfirmString))
	if err != nil {
		return err
	}

	if !crypto.ConfirmECS(pref) {
		// Read unencrypted configuration
		decoder := json.NewDecoder(reader)
		c := &AtRestConfig{}
		err = decoder.Decode(c)
		return err
	}

	conf, err := readEncryptedFileWithKey(reader, keyProvider)
	result, wasEncrypted, err := ReadConfig(confFile, func() ([]byte, error) { return PromptForConfigKey(false) })
	if err != nil {
		return fmt.Errorf("error reading config %w", err)
	}

	json.Unmarshal()
	return nil
}

func Save(atRestPath string) error {
	sbd, err := storage.PrepareForSaving()
	if err != nil {
		return err
	}
	return file.Write(atRestPath, sbd)
}

// readEncryptedConf reads encrypted configuration and requests key from provider
func readEncryptedFileWithKey(reader *bufio.Reader, keyProvider func() ([]byte, error)) (*AtRestConfig, error) {
	fileData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	for errCounter := 0; errCounter < maxAuthFailures; errCounter++ {
		key, err := keyProvider()
		if err != nil {
			log.Errorf(log.ConfigMgr, "PromptForConfigKey err: %s", err)
			continue
		}

		var c *AtRestConfig
		c, err = readEncryptedConf(bytes.NewReader(fileData), key)
		if err != nil {
			log.Errorln(log.ConfigMgr, "Could not decrypt and deserialise data with given key. Invalid password?", err)
			continue
		}
		return c, nil
	}
	return nil, errors.New("failed to decrypt config after 3 attempts")
}

func readEncryptedConf(reader io.Reader, key []byte) (*AtRestConfig, error) {
	data, sessionDK, salt, err := crypto.DecryptFileData(reader, key)
	if err != nil {
		return nil, err
	}
	c := &AtRestConfig{}
	err = json.Unmarshal(data, c)
	c.StoredDK = sessionDK
	c.StoredSalt = salt
	return c, err
}
