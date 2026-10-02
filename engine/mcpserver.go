package engine

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/mcpserver"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

var errMCPConfiguration = errors.New("invalid MCP configuration")

// StartMCPServer binds an authenticated local TLS endpoint only when explicitly enabled.
// It shares RPC handlers directly, so a separate gRPC listener is not required.
func StartMCPServer(ctx context.Context, bot *Engine) error {
	if bot == nil || bot.Config == nil {
		return errMCPConfiguration
	}
	settings := bot.Config.RemoteControl.MCP
	if !settings.Enabled {
		return nil
	}
	host, _, err := net.SplitHostPort(settings.ListenAddress)
	if err != nil {
		return fmt.Errorf("%w: invalid listen address", errMCPConfiguration)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || settings.LogCaptureCapacity < 0 || settings.LogCaptureCapacity > log.MaxCaptureCapacity || settings.RequestTimeoutSeconds < 1 || settings.RequestTimeoutSeconds > 300 || bot.Config.RemoteControl.Username == "" || bot.Config.RemoteControl.Password == "" {
		return fmt.Errorf("%w: require a loopback IP, capacity 0 to 10000, timeout 1 to 300 seconds and remote control credentials", errMCPConfiguration)
	}
	rpc := &RPCServer{Engine: bot}
	server, err := mcpserver.New(rpc, time.Duration(settings.RequestTimeoutSeconds)*time.Second, bot.Config.RemoteControl.GRPC.TimeInNanoSeconds)
	if err != nil {
		return err
	}
	targetDir := utils.GetTLSDir(bot.Settings.DataDir)
	if err := CheckCerts(targetDir); err != nil {
		return fmt.Errorf("MCP certificates: %w", err)
	}
	certificate, err := tls.LoadX509KeyPair(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		return fmt.Errorf("MCP certificate loading: %w", err)
	}
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", settings.ListenAddress)
	if err != nil {
		return fmt.Errorf("MCP listen: %w", err)
	}
	var capture *log.Capture
	if settings.LogCaptureCapacity > 0 {
		capture, err = log.NewCapture(settings.LogCaptureCapacity)
		if err != nil {
			_ = listener.Close()
			return err
		}
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 16384})
	mux := http.NewServeMux()
	mux.Handle("/mcp", rpc.authMCPClient(http.NewCrossOriginProtection().Handler(handler)))
	httpServer := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: time.Duration(settings.RequestTimeoutSeconds+5) * time.Second, IdleTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}, BaseContext: func(net.Listener) context.Context { return ctx }}
	log.DiagnosticCapture.Store(capture)
	bot.ServicesWG.Go(func() {
		stop := context.AfterFunc(ctx, func() { _ = httpServer.Close() })
		defer stop()
		defer log.DiagnosticCapture.Store(nil)
		if err := httpServer.Serve(tls.NewListener(listener, httpServer.TLSConfig)); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			log.Errorf(log.GRPCSys, "MCP server stopped: %s", err)
		}
	})
	log.Infof(log.GRPCSys, "Read-only MCP server enabled on %s", listener.Addr())
	return nil
}

// authMCPClient keeps credentials outside the model's tool arguments.
func (s *RPCServer) authMCPClient(handler http.Handler) http.Handler {
	expectedUser := sha256.Sum256([]byte(s.Config.RemoteControl.Username))
	expectedPassword := sha256.Sum256([]byte(s.Config.RemoteControl.Password))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		userHash := sha256.Sum256([]byte(username))
		passwordHash := sha256.Sum256([]byte(password))
		userMatches := subtle.ConstantTimeCompare(userHash[:], expectedUser[:])
		passwordMatches := subtle.ConstantTimeCompare(passwordHash[:], expectedPassword[:])
		if !ok || userMatches != 1 || passwordMatches != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="gocryptotrader-mcp"`)
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
