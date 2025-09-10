package credentialstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewStore(t *testing.T) {
	t.Parallel()
	s := NewStore()
	if s == nil {
		assert.NotNil(t, s)
	}
}

func TestUpsert(t *testing.T) {
	t.Parallel()
	s := NewStore()
	err := s.Upsert("test", asset.Empty, nil, nil, &account.Credentials{
		Key: "1",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 1)
	err = s.Upsert("test", asset.Spot, nil, nil, &account.Credentials{
		Key: "1",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 2)

	err = s.Upsert("test", asset.Spot, currency.BTC.Item, currency.USD.Item, &account.Credentials{
		Key: "3",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 3)

	err = s.Upsert("test", asset.Spot, currency.BTC.Item, currency.USD.Item, &account.Credentials{
		Key: "3",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 3)

	err = s.Upsert("test", asset.Spot, currency.BTC.Item, currency.USD.Item, &account.Credentials{
		Key: "4",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 3)

	err = s.Upsert("test", asset.Futures, currency.BTC.Item, currency.USD.Item, &account.Credentials{
		Key: "4",
	})
	require.NoError(t, err)
	assert.Len(t, s.exchangePairAssetCredentials, 4)
}
