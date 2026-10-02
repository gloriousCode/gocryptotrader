package config

// MCPConfig controls the read-only MCP endpoint and its optional log history.
// These settings are applied at engine startup; disabled MCP never enables capture.
type MCPConfig struct {
	Enabled               bool   `json:"enabled"`
	ListenAddress         string `json:"listenAddress"`
	LogCaptureCapacity    int    `json:"logCaptureCapacity"`
	RequestTimeoutSeconds int    `json:"requestTimeoutSeconds"`
}
