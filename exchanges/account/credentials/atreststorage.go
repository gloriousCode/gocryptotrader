package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/file"
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

func Load(atRestPath string) error {
	if !file.Exists(atRestPath) {
		return fmt.Errorf("%w %v", common.ErrFileNotFound, atRestPath)
	}
	return nil
}

func Save(atRestPath string) error {
	sbd, err := storage.PrepareForSaving()
	if err != nil {
		return err
	}
	return file.Write(atRestPath, sbd)
}

func Encrypt(key, sessionDK, storedSalt, marshalledKeys []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionDK)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(marshalledKeys))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], marshalledKeys)

	appendedFile := []byte(crypto.EncryptConfirmString)
	appendedFile = append(appendedFile, storedSalt...)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}
