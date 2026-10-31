package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGCCommand(svc Service) *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Remove sessions that have no running container",
		Long: `GC removes sessions left behind when docker ephemeral could not clean up,
for instance after being killed. Sessions with running containers are kept.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			teardowns, err := svc.GC(cmd.Context())
			if len(teardowns) == 0 && err == nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "No stale sessions.")
			}
			printTeardowns(cmd.ErrOrStderr(), teardowns)
			return err
		},
	}
}
