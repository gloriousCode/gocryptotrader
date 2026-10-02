package v16_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v16 "github.com/thrasher-corp/gocryptotrader/config/versions/v16"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		preserved   bool
	}{
		{"previous version", `{"version":15,"remoteControl":{"username":"chosen","gRPC":{"enabled":true}}}`, false},
		{"missing remote control", `{"version":15}`, false},
		{"explicit choice", `{"remoteControl":{"mcp":{"enabled":true,"logCaptureCapacity":7}}}`, true},
		{"explicit null", `{"remoteControl":{"mcp":null}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v16.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "migration must succeed")
			if tc.preserved {
				assert.Equal(t, tc.input, string(got), "migration should preserve explicit selections")
			} else {
				assert.Contains(t, string(got), `"enabled":false`, "migration should disable MCP by default")
				assert.Contains(t, string(got), `"logCaptureCapacity":2000`, "migration should include a bounded capacity")
			}
			again, err := new(v16.Version).UpgradeConfig(t.Context(), got)
			require.NoError(t, err, "repeated migration must succeed")
			assert.Equal(t, got, again, "migration should be idempotent")
		})
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	got, err := new(v16.Version).DowngradeConfig(t.Context(), []byte(`{"remoteControl":{"username":"chosen","mcp":{"enabled":true}},"extra":1}`))
	require.NoError(t, err, "downgrade must succeed")
	assert.JSONEq(t, `{"remoteControl":{"username":"chosen"},"extra":1}`, string(got), "downgrade should remove only MCP")
}

func TestDeployMCPMigration(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":15,"remoteControl":{"username":"chosen","gRPC":{"enabled":true}},"extra":1}`)
	upgraded, err := versions.Manager.Deploy(t.Context(), input, 16)
	require.NoError(t, err, "version manager must upgrade to v16")
	assert.JSONEq(t, `{"version":16,"remoteControl":{"username":"chosen","gRPC":{"enabled":true},"mcp":{"enabled":false,"listenAddress":"127.0.0.1:9054","logCaptureCapacity":2000,"requestTimeoutSeconds":30}},"extra":1}`, string(upgraded), "upgrade should advance version with disabled defaults")
	downgraded, err := versions.Manager.Deploy(t.Context(), upgraded, 15)
	require.NoError(t, err, "version manager must downgrade to v15")
	assert.JSONEq(t, string(input), string(downgraded), "round trip should preserve previous configuration")
}
