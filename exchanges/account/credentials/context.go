package credentials

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc/metadata"
)

const (
	// ContextCredentialsFlag used for retrieving api credentials from context
	ContextCredentialsFlag contextCredential = "apicredentials"
	// ContextSubAccountFlag used for retrieving just the sub account from
	// context, when the default config credentials sub account needs to be
	// changed while the same keys can be used.
	ContextSubAccountFlag contextCredential = "subaccountoverride"
)

// ContextCredentialsStore protects the stored credentials for use in a context
type ContextCredentialsStore struct {
	creds *Credentials
	mu    sync.RWMutex
}

// Load stores provided credentials
func (c *ContextCredentialsStore) Load(creds *Credentials) {
	// Segregate from external call
	cpy := *creds
	c.mu.Lock()
	c.creds = &cpy
	c.mu.Unlock()
}

// Get returns the full credentials from the store
func (c *ContextCredentialsStore) Get() *Credentials {
	c.mu.RLock()
	creds := *c.creds
	c.mu.RUnlock()
	return &creds
}

// ParseCredentialsMetadata intercepts and converts credentials metadata to a
// static type for authentication processing and protection.
func ParseCredentialsMetadata(ctx context.Context, md metadata.MD) (context.Context, error) {
	if md == nil {
		return ctx, errMetaDataIsNil
	}

	credMD, ok := md[string(ContextCredentialsFlag)]
	if !ok || len(credMD) == 0 {
		return ctx, nil
	}

	if len(credMD) != 1 {
		return ctx, errInvalidCredentialMetaDataLength
	}

	segregatedCreds := strings.Split(credMD[0], ",")
	var ctxCreds Credentials
	var subAccountHere string
	for x := range segregatedCreds {
		keyVals := strings.Split(segregatedCreds[x], ":")
		if len(keyVals) != 2 {
			return ctx, fmt.Errorf("%w received %v fields, expected 2 contains: %s",
				errMissingInfo,
				len(keyVals),
				keyVals)
		}
		switch keyVals[0] {
		case Key:
			ctxCreds.Key = keyVals[1]
		case Secret:
			ctxCreds.Secret = keyVals[1]
		case SubAccountSTR:
			// Capture sub account as this can override if other values are
			// not included in metadata.
			subAccountHere = keyVals[1]
		case ClientID:
			ctxCreds.ClientID = keyVals[1]
		case PEMKey:
			ctxCreds.PEMKey = keyVals[1]
		case OneTimePassword:
			ctxCreds.OneTimePassword = keyVals[1]
		}
	}
	if ctxCreds.IsEmpty() && subAccountHere != "" {
		// This will override default sub account details if needed.
		return DeploySubAccountOverrideToContext(ctx, subAccountHere), nil
	}
	// merge sub account to main context credentials
	ctxCreds.SubAccount = subAccountHere
	return DeployCredentialsToContext(ctx, &ctxCreds), nil
}

// DeployCredentialsToContext sets credentials for internal use to context which
// can override default credential values.
func DeployCredentialsToContext(ctx context.Context, creds *Credentials) context.Context {
	flag, store := creds.getInternal()
	return context.WithValue(ctx, flag, store)
}

// DeploySubAccountOverrideToContext sets subaccount as override to credentials
// as a separate flag.
func DeploySubAccountOverrideToContext(ctx context.Context, subAccount string) context.Context {
	return context.WithValue(ctx, ContextSubAccountFlag, subAccount)
}
