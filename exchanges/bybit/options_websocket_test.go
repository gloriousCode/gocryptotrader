package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func countByChannel(subs subscription.List, channel string) int {
	count := 0
	for i := range subs {
		if subs[i].Channel == channel {
			count++
		}
	}
	return count
}

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")
	for i := range subs {
		assert.Equal(t, asset.Options, subs[i].Asset, "Asset type should be Options")
	}

	err = e.CurrencyPairs.SetAssetEnabled(asset.Options, false)
	require.NoError(t, err, "SetAssetEnabled must not error")

	subs, err = e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.Empty(t, subs, "Subscriptions should be empty when asset is disabled")
}

func TestGenerateOptionsDefaultSubscriptionsPublicTradeByUniqueBase(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	pairs := currency.Pairs{
		currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"),
		currency.NewPairWithDelimiter("BTC", "26NOV24-93000-C", "-"),
		currency.NewPairWithDelimiter("ETH", "26NOV24-3500-C", "-"),
	}
	require.NoError(t, e.GetBase().SetPairs(pairs, asset.Options, true), "SetPairs must not error")

	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	require.Equal(t, len(pairs), countByChannel(subs, chanOrderbook), "must subscribe one orderbook topic per options pair")
	require.Equal(t, len(pairs), countByChannel(subs, chanPublicTicker), "must subscribe one ticker topic per options pair")
	require.Equal(t, 2, countByChannel(subs, chanPublicTrade), "must subscribe one public trade topic per unique base")
}

func TestOptionSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")

	err = e.OptionsSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsSubscribe must not error")
}

func TestOptionsUnsubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")

	err = e.OptionsSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsSubscribe must not error")

	err = e.OptionsUnsubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsUnsubscribe must not error")
}

func TestOptionsPublicTradeUsesBaseCoinTopic(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs := subscription.List{
		&subscription.Subscription{
			Channel: chanPublicTrade,
			Pairs:   currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-")},
			Asset:   asset.Options,
		},
	}

	payloads, err := e.directSubscriptionPayload(asset.Options, "subscribe", subs)
	require.NoError(t, err, "directSubscriptionPayload must not error")
	require.Len(t, payloads, 1, "expected a single payload")
	require.Equal(t, []string{"publicTrade.BTC"}, payloads[0].Arguments, "options publicTrade should use baseCoin topic")
}
