package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "test certificate key must generate")
	certificate := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, public, private)
	require.NoError(t, err, "test certificate must generate in memory")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	readFile := func(path string) ([]byte, error) {
		switch path {
		case "policy.json":
			return []byte(`{"remoteControl":{"mcp":{"enabled":true,"mode":"readonly","permissions":{"diagnostics":true}}}}`), nil
		case "testdata/cert.pem":
			return pemData, nil
		case "README.md":
			return []byte("not a certificate"), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	for _, tc := range []struct {
		name     string
		args     []string
		password string
		err      error
		text     string
	}{
		{name: "help", args: []string{"--help"}, text: "Usage of gctmcp"},
		{name: "unknown flag", args: []string{"--unknown"}, text: "flag provided but not defined"},
		{name: "unsupported archive flag", args: []string{"--history-db", "archive.db"}, text: "flag provided but not defined"},
		{name: "positional argument", args: []string{"unexpected"}, err: errUnexpectedArgument},
		{name: "empty host", args: []string{"--rpchost", " "}, err: errMissingRPCAddress},
		{name: "empty username", args: []string{"--rpcuser", " "}, err: errMissingRPCAddress},
		{name: "missing password", err: errMissingPassword},
		{name: "missing certificate", args: []string{"--cert", "testdata/missing.pem"}, password: "test-password", err: os.ErrNotExist},
		{name: "invalid certificate", args: []string{"--cert", "README.md"}, password: "test-password", err: errInvalidCertificate},
		{name: "unsupported capability flag", args: []string{"--allow", "systemControl"}, text: "flag provided but not defined"},
		{name: "invalid timeout", args: []string{"--cert", "testdata/cert.pem", "--timeout", "-1s"}, password: "test-password", text: "timeout"},
		{name: "invalid message limit", args: []string{"--cert", "testdata/cert.pem", "--max-messages", "-1"}, password: "test-password", text: "message"},
		{name: "unsupported offline capability flag", args: []string{"--offline", "--allow", "diagnostics"}, text: "flag provided but not defined"},
		{name: "offline invalid timeout", args: []string{"--offline", "--timeout", "0"}, text: "timeout"},
		{name: "invalid response limit", args: []string{"--cert", "testdata/cert.pem", "--max-response-bytes", "-1"}, password: "test-password", text: "response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			err := run(t.Context(), append([]string{"--config", "policy.json"}, tc.args...), func(key string) string {
				if key == "GCT_RPC_PASSWORD" {
					return tc.password
				}
				return ""
			}, readFile, &stderr, nil)
			if tc.name == "help" {
				require.NoError(t, err, "help must not error")
				assert.Contains(t, stderr.String(), tc.text, "help should list flags")
				return
			}
			require.Error(t, err, "invalid configuration must error")
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "validation should retain the expected error")
			} else {
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tc.text), "validation should identify the invalid setting")
			}
			assert.NotContains(t, err.Error(), "test-password", "errors should not contain the RPC password")
		})
	}

	for _, tc := range []struct {
		name        string
		offline     bool
		policy      string
		available   []string
		unavailable []string
	}{
		{
			name:        "enabled public queries with optional permissions denied",
			policy:      `{"remoteControl":{"mcp":{"enabled":true}}}`,
			available:   []string{"gct_GetTicker", "gct_GetHistoricCandles"},
			unavailable: []string{"gct_runtime_health", "gct_GetAccountBalances", "gct_GetOrders", "gct_SubmitOrder", "gct_EnableExchange", "gct_repair"},
		},
		{
			name:        "readonly gate with explicit optional permissions",
			policy:      `{"remoteControl":{"mcp":{"enabled":true,"mode":"readonly","permissions":{"diagnostics":true,"accountData":true,"orderManagement":true,"withdrawals":true,"systemControl":true}}}}`,
			available:   []string{"gct_runtime_health", "gct_permissions", "gct_GetTicker", "gct_GetAccountBalances", "gct_GetOrders"},
			unavailable: []string{"gct_GetOrder", "gct_SubmitOrder", "gct_CancelOrder", "gct_WithdrawCryptocurrencyFunds", "gct_Shutdown", "gct_EnableExchange", "gct_repair"},
		},
		{
			name:      "execution with explicit permissions",
			policy:    `{"remoteControl":{"mcp":{"enabled":true,"mode":"execution","permissions":{"diagnostics":true,"accountData":true,"orderManagement":true,"withdrawals":true,"systemControl":true}}}}`,
			available: []string{"gct_runtime_health", "gct_GetTicker", "gct_GetAccountBalances", "gct_SubmitOrder", "gct_WithdrawCryptocurrencyFunds", "gct_Shutdown", "gct_EnableExchange", "gct_repair"},
		},
		{
			name: "offline diagnostics need no credentials or certificate", offline: true,
			policy:      `{"remoteControl":{"mcp":{"enabled":true,"permissions":{"diagnostics":true}}}}`,
			available:   []string{"gct_runtime_health", "gct_permissions", "gct_repair_status"},
			unavailable: []string{"gct_GetInfo", "gct_GetTicker", "gct_repair"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			serverTransport, clientTransport := mcp.NewInMemoryTransports()
			args := []string{"--config", "policy.json", "--cert", "testdata/cert.pem", "--rpchost", "localhost:1", "--diagnostics-dir", "unavailable-diagnostics"}
			if tc.offline {
				args = append(args, "--offline", "--rpchost", "", "--rpcuser", "")
			}
			var certificateReads atomic.Int32
			done := make(chan error, 1)
			go func() {
				done <- run(ctx, args, func(key string) string {
					if tc.offline {
						return ""
					}
					if key == "GCT_RPC_PASSWORD" {
						return "test-password"
					}
					return "test-user"
				}, func(path string) ([]byte, error) {
					if path == "policy.json" {
						return []byte(tc.policy), nil
					}
					certificateReads.Add(1)
					return readFile(path)
				}, io.Discard, serverTransport)
			}()
			client := mcp.NewClient(&mcp.Implementation{Name: "gctmcp-cli-test", Version: "test"}, nil)
			session, err := client.Connect(ctx, clientTransport, nil)
			require.NoError(t, err, "MCP client must connect without contacting the RPC server")
			names := make([]string, 0)
			for tool, err := range session.Tools(ctx, nil) {
				require.NoError(t, err, "listing every tool page must not contact the RPC server")
				names = append(names, tool.Name)
			}
			for _, name := range tc.available {
				assert.Contains(t, names, name, "configured and permitted tools should be advertised")
			}
			for _, name := range append(tc.unavailable, "gct_GetExchangeOTPCode", "gct_GetExchangeOTPCodes", "gct_WebsocketSetURL", "gct_WebsocketSetProxy") {
				assert.NotContains(t, names, name, "unavailable tools should not be advertised")
			}
			if tc.offline {
				assert.Zero(t, certificateReads.Load(), "offline startup should not read certificates")
			} else {
				assert.EqualValues(t, 1, certificateReads.Load(), "online startup should read its configured certificate once")
			}
			require.NoError(t, session.Close(), "closing the MCP client must not error")
			select {
			case err := <-done:
				assert.NoError(t, err, "MCP server should stop cleanly")
			case <-ctx.Done():
				require.FailNow(t, "MCP server must stop before the deadline")
			}
		})
	}
}

func TestMainExitStatus(t *testing.T) {
	t.Parallel()
	if args := os.Getenv("GCTMCP_TEST_MAIN_ARGS"); args != "" {
		os.Args = append([]string{"gctmcp"}, strings.Split(args, "\n")...)
		main()
		return
	}
	for _, tc := range []struct {
		name string
		args string
		text string
		fail bool
	}{
		{name: "help", args: "--help", text: "Usage of gctmcp"},
		{name: "invalid argument", args: "unexpected", text: errUnexpectedArgument.Error(), fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainExitStatus$") //nolint:gosec // Re-runs this test binary to exercise main's exit status.
			cmd.Env = append(os.Environ(), "GCTMCP_TEST_MAIN_ARGS="+tc.args)
			out, err := cmd.CombinedOutput()
			if tc.fail {
				assert.Error(t, err, "invalid arguments should produce a non-zero exit status")
			} else {
				assert.NoError(t, err, "help should produce a zero exit status")
			}
			assert.Contains(t, string(out), tc.text, "main should print help or the validation error")
		})
	}
}

func TestReadMCPPolicy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		err  error
	}{
		{name: "enabled public queries", data: `{"remoteControl":{"mcp":{"enabled":true}}}`},
		{name: "diagnostic permission", data: `{"remoteControl":{"mcp":{"enabled":true,"mode":"readonly","permissions":{"diagnostics":true}}}}`},
		{name: "omitted policy", data: `{}`, err: errDisabledPolicy},
		{name: "disabled policy", data: `{"remoteControl":{"mcp":{"enabled":false}}}`, err: errDisabledPolicy},
		{name: "null policy", data: `{"remoteControl":{"mcp":null}}`, err: errDisabledPolicy},
		{name: "malformed policy", data: `{"secret":"private-secret",`, err: errInvalidPolicy},
		{name: "oversized policy", data: strings.Repeat(" ", 4<<20+1), err: errInvalidPolicy},
		{name: "missing file", err: os.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy, err := readMCPPolicy("policy.json", func(path string) ([]byte, error) {
				assert.Equal(t, "policy.json", path, "only the operator-selected policy should be read")
				if tc.data == "" {
					return nil, os.ErrNotExist
				}
				return []byte(tc.data), nil
			})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "invalid policy must fail closed")
				assert.Nil(t, policy, "failed loads should return no authority")
				assert.NotContains(t, err.Error(), "private-secret", "errors should withhold configuration contents")
				return
			}
			require.NoError(t, err, "explicit enabled policy must load")
			assert.True(t, policy.Enabled, "explicitly enabled policy should remain enabled")
			assert.Equal(t, tc.name == "diagnostic permission", policy.Permissions.Diagnostics, "only explicitly granted diagnostics should be enabled")
			assert.False(t, policy.Permissions.AccountData, "omitted account-data permissions should stay denied")
			assert.False(t, policy.Permissions.OrderManagement, "omitted order-management permissions should stay denied")
			assert.False(t, policy.Permissions.Withdrawals, "omitted withdrawal permissions should stay denied")
			assert.False(t, policy.Permissions.SystemControl, "omitted system-control permissions should stay denied")
		})
	}
}
