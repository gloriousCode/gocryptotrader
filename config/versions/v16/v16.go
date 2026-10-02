// Package v16 adds disabled MCP diagnostics without enabling capture for existing installations.
package v16

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version implements ConfigVersion for opt-in MCP diagnostics.
type Version struct{}

// UpgradeConfig preserves an existing MCP selection and adds defaults only when absent.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	if _, _, _, err := jsonparser.Get(config, "remoteControl", "mcp"); err == nil {
		return config, nil
	} else if !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return config, err
	}
	return jsonparser.Set(config, []byte(`{"enabled":false,"listenAddress":"127.0.0.1:9054","logCaptureCapacity":2000,"requestTimeoutSeconds":30}`), "remoteControl", "mcp")
}

// DowngradeConfig removes settings unsupported by the previous config version.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return jsonparser.Delete(config, "remoteControl", "mcp"), nil
}
