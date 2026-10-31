package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newDownCommand(svc Service) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "down [flags] [SESSION...]",
		Short: "Remove sessions and every resource they created",
		Long: `Down removes the given sessions, referenced by name, ID or ID prefix.
Without arguments it removes the only active session.`,
		Example: `  docker ephemeral down
  docker ephemeral down api 3f2a
  docker ephemeral down --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return errors.New("--all cannot be combined with session references")
			}
			teardowns, err := svc.Down(cmd.Context(), args, all)
			printTeardowns(cmd.ErrOrStderr(), teardowns)
			return err
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Remove every session")
	return cmd
}
