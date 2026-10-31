// Package cli exposes the application as docker CLI subcommands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wendersoon/docker-ephemeral/internal/app"
	"github.com/wendersoon/docker-ephemeral/internal/plugin"
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

// NewRootCommand builds the "ephemeral" command tree, nested under a "docker"
// parent so that help and usage read like any other docker subcommand.
func NewRootCommand(svc Service, info BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   plugin.Name,
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
	docker := &cobra.Command{Use: "docker", SilenceUsage: true, SilenceErrors: true}
	docker.CompletionOptions.DisableDefaultCmd = true
	docker.AddCommand(root)
	return root
}

// Execute runs cmd, built by NewRootCommand, with the plugin arguments.
func Execute(ctx context.Context, cmd *cobra.Command, args []string) error {
	root := cmd.Root()
	root.SetArgs(append([]string{cmd.Name()}, args...))
	return root.ExecuteContext(ctx)
}
