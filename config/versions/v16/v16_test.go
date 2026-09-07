package v16_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v16 "github.com/thrasher-corp/gocryptotrader/config/versions/v16"
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	version := new(v16.Version)
	got, err := version.UpgradeExchange(t.Context(), nil)
	require.NoError(t, err, "UpgradeExchange must not error for empty input")
	assert.Nil(t, got, "UpgradeExchange should preserve empty input")

	payload := []byte(`{"name":"test"}`)
	got, err = version.UpgradeExchange(t.Context(), payload)
	require.NoError(t, err, "UpgradeExchange must add the field")
	assert.JSONEq(t, `{"name":"test","websocketMetricsLogging":false}`, string(got), "UpgradeExchange should default metrics logging to false")

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = version.UpgradeExchange(t.Context(), payload)
	require.NoError(t, err, "UpgradeExchange must preserve an existing field")
	assert.Equal(t, payload, got, "UpgradeExchange should preserve an existing value")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err := new(v16.Version).DowngradeExchange(t.Context(), payload)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.JSONEq(t, `{"name":"test"}`, string(got), "DowngradeExchange should remove metrics logging")
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v16.Version).Exchanges(), "Exchanges should select every exchange")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":15,"exchanges":[{"name":"GateIO"}]}`)
	got, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "Deploy must apply the registered v16 upgrade")
	assert.JSONEq(t, `{"version":16,"exchanges":[{"name":"GateIO","websocketMetricsLogging":false}]}`, string(got), "Deploy should add metrics logging and set version 16")
}
