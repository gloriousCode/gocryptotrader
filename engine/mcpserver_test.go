package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStartMCPServer(t *testing.T) {
	original := gctlog.DiagnosticCapture.Load()
	gctlog.DiagnosticCapture.Store(nil)
	t.Cleanup(func() { gctlog.DiagnosticCapture.Store(original) })
	require.ErrorIs(t, StartMCPServer(t.Context(), nil), errMCPConfiguration, "nil engine must fail")
	require.ErrorIs(t, StartMCPServer(t.Context(), &Engine{}), errMCPConfiguration, "missing config must fail")
	bot := &Engine{Config: &config.Config{RemoteControl: config.RemoteControlConfig{MCP: config.MCPConfig{Enabled: false, LogCaptureCapacity: 2000}}}}
	require.NoError(t, StartMCPServer(t.Context(), bot), "disabled MCP must skip setup")
	assert.Nil(t, gctlog.DiagnosticCapture.Load(), "disabled MCP should allocate no capture history")
	for _, tc := range []struct {
		name, address     string
		capacity, timeout int
	}{
		{"bad address", "bad", 1, 30},
		{"public binding", "0.0.0.0:9054", 1, 30},
		{"hostname", "localhost:9054", 1, 30},
		{"bad capacity", "127.0.0.1:9054", 10001, 30},
		{"negative capacity", "127.0.0.1:9054", -1, 30},
		{"bad timeout", "127.0.0.1:9054", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bot := &Engine{Config: &config.Config{RemoteControl: config.RemoteControlConfig{Username: "user", Password: "placeholder", MCP: config.MCPConfig{Enabled: true, ListenAddress: tc.address, LogCaptureCapacity: tc.capacity, RequestTimeoutSeconds: tc.timeout}}}}
			require.ErrorIs(t, StartMCPServer(t.Context(), bot), errMCPConfiguration, "invalid settings must fail before startup")
			assert.Nil(t, gctlog.DiagnosticCapture.Load(), "failed startup should not enable capture")
		})
	}
}

func TestAuthMCPClient(t *testing.T) {
	t.Parallel()
	rpc := &RPCServer{Engine: &Engine{Config: &config.Config{RemoteControl: config.RemoteControlConfig{Username: "user", Password: "placeholder"}}}}
	for _, tc := range []struct {
		name, user, password, origin string
		status                       int
	}{
		{name: "no auth", status: http.StatusUnauthorized},
		{name: "wrong user", user: "wrong", password: "placeholder", status: http.StatusUnauthorized},
		{name: "wrong password", user: "user", password: "wrong", status: http.StatusUnauthorized},
		{name: "authorised", user: "user", password: "placeholder", status: http.StatusNoContent},
		{name: "cross origin", user: "user", password: "placeholder", origin: "https://evil.example", status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := rpc.authMCPClient(http.NewCrossOriginProtection().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://localhost/mcp", http.NoBody)
			if tc.user != "" {
				r.SetBasicAuth(tc.user, tc.password)
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, tc.status, w.Code, "access control should enforce credentials and origin")
		})
	}
}

func TestGetDiagnosticLogs(t *testing.T) {
	original := gctlog.DiagnosticCapture.Load()
	t.Cleanup(func() { gctlog.DiagnosticCapture.Store(original) })
	gctlog.DiagnosticCapture.Store(nil)
	rpc := &RPCServer{}
	disabled, err := rpc.GetDiagnosticLogs(t.Context(), &gctrpc.GetDiagnosticLogsRequest{})
	require.NoError(t, err, "disabled query must succeed")
	assert.False(t, disabled.Enabled, "disabled capture should be explicit")
	for _, r := range []*gctrpc.GetDiagnosticLogsRequest{nil, {Limit: 201}, {Severity: "fatal"}, {Contains: string(make([]byte, 257))}, {Since: &timestamppb.Timestamp{Nanos: -1}}, {Until: &timestamppb.Timestamp{Seconds: 999999999999}}, {Since: timestamppb.New(time.Unix(2, 0)), Until: timestamppb.New(time.Unix(1, 0))}} {
		_, err := rpc.GetDiagnosticLogs(t.Context(), r)
		require.Error(t, err, "invalid query must fail")
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "invalid query should have a stable RPC status")
	}
	c, err := gctlog.NewCapture(2)
	require.NoError(t, err, "capture must initialise")
	c.Record(time.Unix(1, 0), "error", "WEBSOCKET", "failed")
	c.Record(time.Unix(2, 0), "error", "WEBSOCKET", "failed")
	gctlog.DiagnosticCapture.Store(c)
	logs, err := rpc.GetDiagnosticLogs(t.Context(), &gctrpc.GetDiagnosticLogsRequest{Limit: 1})
	require.NoError(t, err, "query must succeed")
	require.Len(t, logs.Entries, 1, "pagination must bound entries")
	assert.True(t, logs.HasMore, "pagination should report more evidence")
	summary, err := rpc.GetDiagnosticLogs(t.Context(), &gctrpc.GetDiagnosticLogsRequest{Summary: true, Since: timestamppb.New(time.Unix(1, 0)), Until: timestamppb.New(time.Unix(2, 0))})
	require.NoError(t, err, "summary must succeed")
	require.Len(t, summary.Groups, 1, "summary must group repeats")
	assert.Equal(t, uint64(2), summary.Groups[0].Count, "summary should cover the complete retained window")
}

func TestMCPServerTLSLifecycle(t *testing.T) {
	original := gctlog.DiagnosticCapture.Load()
	t.Cleanup(func() { gctlog.DiagnosticCapture.Store(original) })
	// This committed certificate and key are public test fixtures, never runtime credentials.
	cert, err := os.ReadFile("testdata/mcp/tls/cert.pem")
	require.NoError(t, err, "test certificate must load")
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(cert), "test certificate must be trusted explicitly")
	for _, capacity := range []int{0, 10} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err, "test port must be available")
			address := listener.Addr().String()
			require.NoError(t, listener.Close(), "test port reservation must close")
			ctx, cancel := context.WithCancel(t.Context())
			bot := &Engine{Settings: Settings{DataDir: "testdata/mcp"}, Config: &config.Config{RemoteControl: config.RemoteControlConfig{Username: "user", Password: "placeholder", MCP: config.MCPConfig{Enabled: true, ListenAddress: address, LogCaptureCapacity: capacity, RequestTimeoutSeconds: 2}}}}
			require.NoError(t, StartMCPServer(ctx, bot), "enabled MCP must start with fixture TLS")
			defer func() { cancel(); bot.ServicesWG.Wait() }()
			if capacity == 0 {
				assert.Nil(t, gctlog.DiagnosticCapture.Load(), "zero capacity should disable capture")
			} else {
				require.NotNil(t, gctlog.DiagnosticCapture.Load(), "positive capacity must install capture")
				gctlog.DiagnosticCapture.Load().Record(time.Now(), "error", "WEBSOCKET", "test failure")
			}
			transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
			for _, query := range []struct {
				body   string
				auth   bool
				status int
			}{
				{`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, false, http.StatusUnauthorized},
				{`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, true, http.StatusOK},
				{`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_recent_logs","arguments":{}}}`, true, http.StatusOK},
				{strings.Repeat("x", 16385), true, http.StatusRequestEntityTooLarge},
			} {
				request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://"+address+"/mcp", strings.NewReader(query.body))
				require.NoError(t, err, "MCP request must initialise")
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set("MCP-Protocol-Version", "2025-06-18")
				if query.auth {
					request.SetBasicAuth("user", "placeholder")
				}
				response, err := client.Do(request)
				require.NoError(t, err, "TLS MCP request must succeed")
				data, err := io.ReadAll(response.Body)
				require.NoError(t, err, "MCP response must be readable")
				require.NoError(t, response.Body.Close(), "response body must close")
				assert.Equal(t, query.status, response.StatusCode, "MCP should enforce auth and body limits")
				if query.status == http.StatusOK {
					assert.Contains(t, string(data), `"result"`, "MCP should return a protocol result")
				}
			}
			cancel()
			bot.ServicesWG.Wait()
			assert.Nil(t, gctlog.DiagnosticCapture.Load(), "shutdown should release capture")
		})
	}
}
