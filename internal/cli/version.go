package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the docker ephemeral version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "docker ephemeral %s (commit %s)\n", info.Version, info.Commit)
		},
	}
}
