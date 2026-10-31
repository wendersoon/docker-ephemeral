package cli

import (
	"errors"
	"slices"

	"github.com/spf13/cobra"
)

func newRunCommand(svc Service) *cobra.Command {
	return &cobra.Command{
		Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Run a container and remove everything it created when it exits",
		Long: `Run accepts the same options as "docker run" and runs the container in the
foreground. When it exits, the container, its volumes and any image pulled
for it are removed. The exit code of the container is preserved.`,
		Example: `  docker ephemeral run --rm -it alpine sh
  docker ephemeral run -p 8080:80 -v data:/data nginx`,
		// Every argument belongs to docker run.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && slices.Contains([]string{"-h", "--help"}, args[0]) {
				return cmd.Help()
			}
			if len(args) == 0 {
				return errors.New(`"run" requires at least 1 argument: IMAGE`)
			}

			res, err := svc.Run(cmd.Context(), args)
			if res.Session != nil {
				printTeardown(cmd.ErrOrStderr(), res.Teardown)
			}
			if err != nil {
				return err
			}
			if res.ExitCode != 0 {
				return &ExitError{Code: res.ExitCode}
			}
			return nil
		},
	}
}
