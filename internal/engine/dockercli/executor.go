// Package dockercli implements engine.Engine on top of the docker CLI.
package dockercli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
)

// Command describes a docker CLI invocation.
type Command struct {
	Args []string
	// Attached connects the process to the terminal and reports its exit
	// code instead of failing on non-zero status.
	Attached bool
	// ShowProgress streams stderr to the user while still capturing it.
	ShowProgress bool
}

// Result is the outcome of a Command.
type Result struct {
	Stdout   []byte
	ExitCode int
}

// Executor runs docker CLI commands.
type Executor interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

// CommandError reports a failed docker invocation.
type CommandError struct {
	Args   []string
	Stderr string
	kind   error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("docker %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

// Unwrap exposes engine.ErrNotFound or engine.ErrInUse when applicable.
func (e *CommandError) Unwrap() error { return e.kind }

// ProcessExecutor runs the docker binary as a child process.
type ProcessExecutor struct {
	Binary      string
	GlobalFlags []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

const shutdownGrace = 10 * time.Second

// NewProcessExecutor returns an executor bound to the process standard streams.
func NewProcessExecutor(binary string, globalFlags []string) *ProcessExecutor {
	return &ProcessExecutor{
		Binary:      binary,
		GlobalFlags: globalFlags,
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}
}

// Execute implements Executor.
func (p *ProcessExecutor) Execute(ctx context.Context, c Command) (Result, error) {
	cmd := exec.CommandContext(ctx, p.Binary, slices.Concat(p.GlobalFlags, c.Args)...)
	if c.Attached {
		return p.attach(cmd)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if c.ShowProgress {
		cmd.Stderr = io.MultiWriter(&stderr, p.Stderr)
	}

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return Result{}, fmt.Errorf("run docker: %w", err)
		}
		msg := strings.TrimSpace(stderr.String())
		return Result{ExitCode: exitErr.ExitCode()}, &CommandError{Args: c.Args, Stderr: msg, kind: classify(msg)}
	}
	return Result{Stdout: stdout.Bytes()}, nil
}

// attach runs cmd in the foreground. The child shares our process group, so
// the terminal already delivers SIGINT to it; we only swallow it to survive
// long enough to clean up, and forward termination signals sent to us alone.
func (p *ProcessExecutor) attach(cmd *exec.Cmd) (Result, error) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = p.Stdin, p.Stdout, p.Stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = shutdownGrace

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("run docker: %w", err)
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case sig := <-signals:
				if sig != os.Interrupt {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	err := cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return Result{ExitCode: exitErr.ExitCode()}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("run docker: %w", err)
	}
	return Result{}, nil
}

func classify(stderr string) error {
	msg := strings.ToLower(stderr)
	switch {
	case strings.Contains(msg, "no such"), strings.Contains(msg, "not found"):
		return engine.ErrNotFound
	case strings.Contains(msg, "in use"),
		strings.Contains(msg, "being used"),
		strings.Contains(msg, "active endpoints"),
		strings.Contains(msg, "conflict"):
		return engine.ErrInUse
	default:
		return nil
	}
}
