package credentials

import (
	"errors"
	"fmt"
	"strings"
)

// contextCredential is a string flag for use with context values when setting
// credentials internally or via gRPC.
type contextCredential string

const (
	apiKeyDisplaySize = 16
)

// Default credential values
const (
	Key             = "key"
	Secret          = "secret"
	SubAccountSTR   = "subaccount"
	ClientID        = "clientid"
	OneTimePassword = "otp"
	PEMKey          = "pemkey"
)

var (
	errMetaDataIsNil                   = errors.New("meta data is nil")
	errInvalidCredentialMetaDataLength = errors.New("invalid meta data to process credentials")
	errMissingInfo                     = errors.New("cannot parse meta data missing information in key value pair")
)

// Credentials define parameters that allow for an authenticated request.
type Credentials struct {
	Key                 string
	Secret              string
	ClientID            string // TODO: Implement with exchange orders functionality
	PEMKey              string
	SubAccount          string
	OneTimePassword     string
	SecretBase64Decoded bool
	// TODO: Add AccessControl uint8 for READ/WRITE/Withdraw capabilities.
}

// GetMetaData returns the credentials for metadata context deployment
func (c *Credentials) GetMetaData() (flag, values string) {
	vals := make([]string, 0, 6)
	if c.Key != "" {
		vals = append(vals, Key+":"+c.Key)
	}
	if c.Secret != "" {
		vals = append(vals, Secret+":"+c.Secret)
	}
	if c.SubAccount != "" {
		vals = append(vals, SubAccountSTR+":"+c.SubAccount)
	}
	if c.ClientID != "" {
		vals = append(vals, ClientID+":"+c.ClientID)
	}
	if c.PEMKey != "" {
		vals = append(vals, PEMKey+":"+c.PEMKey)
	}
	if c.OneTimePassword != "" {
		vals = append(vals, OneTimePassword+":"+c.OneTimePassword)
	}
	return string(ContextCredentialsFlag), strings.Join(vals, ",")
}

// String prints out basic credential info (obfuscated) to track key instances
// associated with exchanges.
func (c *Credentials) String() string {
	obfuscated := c.Key
	if len(obfuscated) > apiKeyDisplaySize {
		obfuscated = obfuscated[:apiKeyDisplaySize]
	}
	return fmt.Sprintf("Key:[%s...] SubAccount:[%s] ClientID:[%s]",
		obfuscated,
		c.SubAccount,
		c.ClientID)
}

// getInternal returns the values for assignment to an internal context
func (c *Credentials) getInternal() (contextCredential, *ContextCredentialsStore) {
	if c.IsEmpty() {
		return "", nil
	}
	store := &ContextCredentialsStore{}
	store.Load(c)
	return ContextCredentialsFlag, store
}

// IsEmpty return true if the underlying credentials type has not been filled
// with at least one item.
func (c *Credentials) IsEmpty() bool {
	return c == nil || c.ClientID == "" &&
		c.Key == "" &&
		c.OneTimePassword == "" &&
		c.PEMKey == "" &&
		c.Secret == "" &&
		c.SubAccount == ""
}

// Equal determines if the keys are the same.
// OTP omitted because it's generated per request.
// PEMKey and Secret omitted because of direct correlation with api key.
func (c *Credentials) Equal(other *Credentials) bool {
	return c != nil &&
		other != nil &&
		c.Key == other.Key &&
		c.ClientID == other.ClientID &&
		(c.SubAccount == other.SubAccount || c.SubAccount == "" && other.SubAccount == "main" || c.SubAccount == "main" && other.SubAccount == "")
}
