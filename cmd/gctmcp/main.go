// gctmcp serves configured GCT commands and saved diagnostics over MCP stdio.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/internal/gctmcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	errUnexpectedArgument = errors.New("unexpected positional argument; use named flags")
	errMissingPassword    = errors.New("GCT_RPC_PASSWORD must contain the RPC server password")
	errMissingRPCAddress  = errors.New("RPC host and username must not be empty")
	errInvalidCertificate = errors.New("RPC TLS certificate must contain a valid PEM certificate")
	errInvalidPolicy      = errors.New("MCP policy must be unencrypted JSON containing remoteControl.mcp")
	errDisabledPolicy     = errors.New("MCP is disabled in remoteControl.mcp.enabled")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Getenv, os.ReadFile, os.Stderr, &mcp.StdioTransport{})
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, readFile func(string) ([]byte, error), stderr io.Writer, transport mcp.Transport) error {
	flags := flag.NewFlagSet("gctmcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	host := flags.String("rpchost", "localhost:9052", "GCT gRPC server address")
	username := getenv("GCT_RPC_USERNAME")
	if username == "" {
		username = "admin"
	}
	flags.StringVar(&username, "rpcuser", username, "GCT RPC username (defaults to GCT_RPC_USERNAME or admin)")
	dataDir := common.GetDefaultDataDir(runtime.GOOS)
	configPath := flags.String("config", filepath.Join(dataDir, "config.json"), "GCT JSON configuration or a separate MCP policy JSON file")
	cert := flags.String("cert", filepath.Join(dataDir, "tls", "cert.pem"), "GCT RPC server TLS certificate")
	opts := gctmcp.Options{}
	flags.StringVar(&opts.DiagnosticsDir, "diagnostics-dir", filepath.Join(dataDir, "diagnostics"), "GCT diagnostic journal directory")
	flags.BoolVar(&opts.Offline, "offline", false, "inspect saved diagnostics without RPC credentials or a TLS certificate")
	flags.DurationVar(&opts.Timeout, "timeout", 10*time.Second, "maximum duration of each RPC command")
	flags.IntVar(&opts.MaxMessages, "max-messages", 20, "maximum messages collected from an RPC stream")
	flags.IntVar(&opts.MaxResponseBytes, "max-response-bytes", 1<<20, "maximum encoded response size")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errUnexpectedArgument
	}
	policy, err := readMCPPolicy(*configPath, readFile)
	if err != nil {
		return err
	}
	opts.Policy = *policy
	var conn *grpc.ClientConn
	if !opts.Offline {
		if strings.TrimSpace(*host) == "" || strings.TrimSpace(username) == "" {
			return errMissingRPCAddress
		}
		password := getenv("GCT_RPC_PASSWORD")
		if password == "" {
			return errMissingPassword
		}
		certificate, err := readFile(*cert)
		if err != nil {
			return fmt.Errorf("load RPC TLS certificate: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(certificate) {
			return errInvalidCertificate
		}
		conn, err = grpc.NewClient(*host,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12})),
			grpc.WithPerRPCCredentials(auth.BasicAuth{Username: username, Password: password}),
			grpc.WithDisableRetry(),
		)
		if err != nil {
			return fmt.Errorf("configure RPC connection: %w", err)
		}
		defer conn.Close()
	}
	bridge, err := gctmcp.New(conn, &opts)
	if err != nil {
		return err
	}
	if err := bridge.Server().Run(ctx, transport); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("serve MCP: %w", err)
	}
	return nil
}

// readMCPPolicy leaves protocol input dedicated to MCP messages.
func readMCPPolicy(path string, readFile func(string) ([]byte, error)) (*config.MCPConfig, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("load MCP policy: %w", err)
	}
	var document struct {
		RemoteControl struct {
			MCP config.MCPConfig `json:"mcp"`
		} `json:"remoteControl"`
	}
	if len(data) > 4<<20 || json.Unmarshal(data, &document) != nil {
		return nil, errInvalidPolicy
	}
	if !document.RemoteControl.MCP.Enabled {
		return nil, errDisabledPolicy
	}
	return &document.RemoteControl.MCP, nil
}
