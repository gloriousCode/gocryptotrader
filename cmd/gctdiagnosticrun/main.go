package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/internal/diagnostics"
)

var (
	errMissingExecutable = errors.New("an executable after -- is required")
	errCommandSeparator  = errors.New("separate launcher options from the executable with --")
	errDiagnosticsPath   = errors.New("diagnostic directory must not be empty")
	errSupervisorLocked  = errors.New("diagnostic supervisor lock already exists; reconcile the previous launcher and child before removing it")
	errChildStart        = errors.New("diagnostic child failed to start")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, diagnostics.Write)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, writeSnapshot func(string, *diagnostics.Snapshot) error) (resultErr error) {
	flags := flag.NewFlagSet("gctdiagnosticrun", flag.ContinueOnError)
	flags.SetOutput(stderr)
	directory := flags.String("diagnostics-dir", filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "diagnostics"), "GCT diagnostic journal directory")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() == 0 {
		return errMissingExecutable
	}
	if !slices.Contains(args, "--") {
		return errCommandSeparator
	}
	if strings.TrimSpace(*directory) == "" {
		return errDiagnosticsPath
	}
	if err := os.MkdirAll(*directory, 0o700); err != nil {
		return fmt.Errorf("create diagnostic directory: %w", err)
	}
	if err := os.Chmod(*directory, 0o700); err != nil {
		return fmt.Errorf("protect diagnostic directory: %w", err)
	}
	lockPath := filepath.Join(*directory, "supervisor.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errSupervisorLocked
		}
		return fmt.Errorf("lock diagnostic supervisor: %w", err)
	}
	defer func() {
		if err := os.Remove(lockPath); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("release diagnostic supervisor lock: %w", err))
		}
	}()
	if _, err := fmt.Fprintln(lock, os.Getpid()); err != nil {
		_ = lock.Close()
		return fmt.Errorf("write diagnostic supervisor lock: %w", err)
	}
	if err := lock.Sync(); err != nil {
		_ = lock.Close()
		return fmt.Errorf("sync diagnostic supervisor lock: %w", err)
	}
	if err := lock.Close(); err != nil {
		return fmt.Errorf("close diagnostic supervisor lock: %w", err)
	}
	started := time.Now().UTC()
	record := &crashRecord{StartedAt: started}
	snapshot := &diagnostics.Snapshot{Version: 1, RunID: rand.Text(), StartedAt: started, CapturedAt: started, State: "running", Views: map[string]any{"crash_report": record}}
	path := filepath.Join(*directory, "crash.json")
	if err := writeSnapshot(path, snapshot); err != nil {
		return fmt.Errorf("record child launch intent: %w", err)
	}
	childContext, cancel := context.WithCancel(ctx)
	defer cancel()
	child := exec.CommandContext(childContext, flags.Arg(0), flags.Args()[1:]...) //nolint:gosec // Only the operator's explicit launcher arguments select this child; MCP never calls the launcher.
	child.Cancel = func() error { return child.Process.Signal(os.Interrupt) }
	child.WaitDelay = 10 * time.Second
	child.Stdout = stdout
	observed := &panicWriter{output: stderr}
	child.Stderr = observed
	if err := child.Start(); err != nil {
		record.StartFailed = true
		record.Interrupted = ctx.Err() != nil
		record.FinishedAt = time.Now().UTC()
		snapshot.State, snapshot.CapturedAt = "unclean", record.FinishedAt
		writeErr := writeSnapshot(path, snapshot)
		return errors.Join(fmt.Errorf("%w: %w", errChildStart, err), writeErr)
	}
	record.ProcessID = child.Process.Pid
	snapshot.CapturedAt = time.Now().UTC()
	if err := writeSnapshot(path, snapshot); err != nil {
		cancel()
		return errors.Join(fmt.Errorf("record running child: %w", err), child.Wait())
	}
	waitErr := child.Wait()
	record.FinishedAt = time.Now().UTC()
	record.PanicObserved = observed.seen
	record.Interrupted = ctx.Err() != nil
	record.ExitCode = new(child.ProcessState.ExitCode())
	snapshot.State, snapshot.CapturedAt = "stopped", record.FinishedAt
	if *record.ExitCode != 0 {
		snapshot.State = "unclean"
	}
	if err := writeSnapshot(path, snapshot); err != nil {
		return errors.Join(waitErr, fmt.Errorf("record child completion: %w", err))
	}
	return waitErr
}

// Write observes a bounded panic marker while preserving the downstream output.
func (p *panicWriter) Write(data []byte) (int, error) {
	const marker = "panic:"
	text := p.tail + string(data)
	p.seen = p.seen || strings.Contains(text, marker)
	p.tail = text[max(0, len(text)-len(marker)+1):]
	return p.output.Write(data)
}
