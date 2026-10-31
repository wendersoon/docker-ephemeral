// Package cli exposes the application as docker CLI subcommands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wendersoon/docker-ephemeral/internal/app"
)

// Service is the application API used by the commands.
type Service interface {
	Run(ctx context.Context, args []string) (app.RunResult, error)
	Up(ctx context.Context, req app.UpRequest) (app.UpResult, error)
	Down(ctx context.Context, refs []string, all bool) ([]app.Teardown, error)
	List(ctx context.Context) ([]app.Status, error)
	GC(ctx context.Context) ([]app.Teardown, error)
}

// BuildInfo identifies the running binary.
type BuildInfo struct {
	Version string
	Commit  string
}

// ExitError carries the exit status of a workload back to the process.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

// NewRootCommand builds the command tree.
func NewRootCommand(svc Service, info BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   "docker ephemeral",
		Short: "Run Docker workloads that leave no trace behind",
		Long: `docker ephemeral runs containers and Compose projects inside a session.
When the session ends, every container, network, volume and image it
created is removed. Resources that existed before are never touched.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(
		newRunCommand(svc),
		newUpCommand(svc),
		newDownCommand(svc),
		newListCommand(svc),
		newGCCommand(svc),
		newVersionCommand(info),
	)
	return root
}
