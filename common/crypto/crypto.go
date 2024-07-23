package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // Used for exchanges
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Used for exchanges
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"io"

	"golang.org/x/crypto/scrypt"
)

// Const declarations for common.go operations
const (
	HashSHA1 = iota
	HashSHA256
	HashSHA512
	HashSHA512_384
	HashMD5
)

const (
	// EncryptConfirmString has a the general confirmation string to allow us to
	// see if the file is correctly encrypted
	EncryptConfirmString = "THORS-HAMMER"
	// SaltPrefix string
	SaltPrefix = "~GCT~SO~SALTY~"
	// SaltRandomLength is the number of random bytes to append after the prefix string
	SaltRandomLength = 12
)

var (
	errAESBlockSize = "config file data is too small for the AES required block size"
)

// HexEncodeToString takes in a hexadecimal byte array and returns a string
func HexEncodeToString(input []byte) string {
	return hex.EncodeToString(input)
}

// Base64Decode takes in a Base64 string and returns a byte array and an error
func Base64Decode(input string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Base64Encode takes in a byte array then returns an encoded base64 string
func Base64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

// GetRandomSalt returns a random salt
func GetRandomSalt(input []byte, saltLen int) ([]byte, error) {
	if saltLen <= 0 {
		return nil, errors.New("salt length is too small")
	}
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	var result []byte
	if input != nil {
		result = input
	}
	result = append(result, salt...)
	return result, nil
}

// GetMD5 returns a MD5 hash of a byte array
func GetMD5(input []byte) ([]byte, error) {
	m := md5.New() //nolint:gosec // hash function used by some exchanges
	_, err := m.Write(input)
	return m.Sum(nil), err
}

// GetSHA512 returns a SHA512 hash of a byte array
func GetSHA512(input []byte) ([]byte, error) {
	sha := sha512.New()
	_, err := sha.Write(input)
	return sha.Sum(nil), err
}

// GetSHA256 returns a SHA256 hash of a byte array
func GetSHA256(input []byte) ([]byte, error) {
	sha := sha256.New()
	_, err := sha.Write(input)
	return sha.Sum(nil), err
}

// GetHMAC returns a keyed-hash message authentication code using the desired
// hashtype
func GetHMAC(hashType int, input, key []byte) ([]byte, error) {
	var hasher func() hash.Hash

	switch hashType {
	case HashSHA1:
		hasher = sha1.New
	case HashSHA256:
		hasher = sha256.New
	case HashSHA512:
		hasher = sha512.New
	case HashSHA512_384:
		hasher = sha512.New384
	case HashMD5:
		hasher = md5.New
	}

	h := hmac.New(hasher, key)
	_, err := h.Write(input)
	return h.Sum(nil), err
}

// Sha1ToHex takes a string, sha1 hashes it and return a hex string of the
// result
func Sha1ToHex(data string) (string, error) {
	h := sha1.New() //nolint:gosec // hash function used by some exchanges
	_, err := h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil)), err
}

func MakeNewSessionDK(key []byte) (dk, storedSalt []byte, err error) {
	storedSalt, err = GetRandomSalt([]byte(SaltPrefix), SaltRandomLength)
	if err != nil {
		return nil, nil, err
	}

	dk, err = GetScryptDK(key, storedSalt)
	if err != nil {
		return nil, nil, err
	}

	return dk, storedSalt, nil
}

// ConfirmSalt checks whether the encrypted data contains a salt
func ConfirmSalt(file []byte) bool {
	return bytes.Contains(file, []byte(SaltPrefix))
}

// ConfirmECS confirms that the encryption confirmation string is found
func ConfirmECS(file []byte) bool {
	return bytes.Contains(file, []byte(EncryptConfirmString))
}

// SkipECS skips encryption confirmation string
// or errors, if the prefix wasn't found
func SkipECS(file io.Reader) error {
	buf := make([]byte, len(EncryptConfirmString))
	if _, err := io.ReadFull(file, buf); err != nil {
		return err
	}
	if string(buf) != EncryptConfirmString {
		return errors.New("data does not start with ECS")
	}
	return nil
}

func GetScryptDK(key, salt []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("key is empty")
	}
	return scrypt.Key(key, salt, 32768, 8, 1, 32)
}

func HandleSessionEncryptData(data, key []byte) (eData, sessionDK, salt []byte, err error) {
	sessionDK, salt, err = MakeNewSessionDK(key)
	if err != nil {
		return nil, nil, nil, err
	}
	eData, err = encryptData(data, sessionDK, salt)
	if err != nil {
		return nil, nil, nil, err
	}
	return eData, sessionDK, salt, nil
}

func encryptData(data, sessionDK, storedSalt []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionDK)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	appendedFile := []byte(EncryptConfirmString)
	appendedFile = append(appendedFile, storedSalt...)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

func DecryptFileData(fileReader io.Reader, key []byte) (fileData, sessionDK, salt []byte, err error) {
	err = SkipECS(fileReader)
	if err != nil {
		return nil, nil, nil, err
	}
	origKey := key
	configData, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, nil, nil, err
	}

	if ConfirmSalt(configData) {
		salt = make([]byte, len(SaltPrefix)+SaltRandomLength)
		salt = configData[0:len(salt)]
		key, err = GetScryptDK(key, salt)
		if err != nil {
			return nil, nil, nil, err
		}
		configData = configData[len(salt):]
	}

	blockDecrypt, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(configData) < aes.BlockSize {
		return nil, nil, nil, errors.New(errAESBlockSize)
	}

	iv := configData[:aes.BlockSize]
	configData = configData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(blockDecrypt, iv)
	stream.XORKeyStream(configData, configData)
	result := configData

	sessionDK, salt, err = MakeNewSessionDK(origKey)
	if err != nil {
		return nil, nil, nil, err
	}
	return result, sessionDK, salt, nil
}
