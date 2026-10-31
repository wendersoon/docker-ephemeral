package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newListCommand(svc Service) *cobra.Command {
	var quiet bool

	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List active sessions",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			statuses, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if quiet {
				for _, st := range statuses {
					fmt.Fprintln(out, st.Session.ID)
				}
				return nil
			}

			now := time.Now()
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "SESSION ID\tNAME\tMODE\tCONTAINERS\tCREATED")
			for _, st := range statuses {
				s := st.Session
				fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d running\t%s\n",
					s.ID, s.Name, s.Mode, st.Running, st.Containers, humanizeSince(s.CreatedAt, now))
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display session IDs")
	return cmd
}
