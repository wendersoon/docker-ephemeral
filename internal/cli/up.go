package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wendersoon/docker-ephemeral/internal/app"
)

func newUpCommand(svc Service) *cobra.Command {
	var req app.UpRequest

	cmd := &cobra.Command{
		Use:   "up [flags] [SERVICE...] [-- COMPOSE_UP_ARGS...]",
		Short: "Start a Compose project as an ephemeral session",
		Long: `Up starts a Docker Compose project under a dedicated project name and
prints the session ID. Everything the project creates is removed by
"docker ephemeral down".

With --attach, logs are followed and the session is removed as soon as you
press Ctrl+C. Arguments after "--" are forwarded to "docker compose up".`,
		Example: `  docker ephemeral up
  docker ephemeral up -f compose.yml --name api -- --build
  docker ephemeral up --attach web db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req.Args = args
			res, err := svc.Up(cmd.Context(), req)
			if res.Destroyed {
				printTeardown(cmd.ErrOrStderr(), res.Teardown)
			}
			if err != nil {
				return err
			}
			if !res.Destroyed {
				fmt.Fprintln(cmd.OutOrStdout(), res.Session.ID)
				fmt.Fprintf(cmd.ErrOrStderr(), "Session %s is up. Remove it with: docker ephemeral down %s\n",
					res.Session.DisplayName(), res.Session.ID)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&req.Name, "name", "n", "", "Session name")
	flags.BoolVarP(&req.Attach, "attach", "a", false, "Follow logs and remove the session on exit")
	flags.StringArrayVarP(&req.Compose.Files, "file", "f", nil, "Compose configuration files")
	flags.StringArrayVar(&req.Compose.EnvFiles, "env-file", nil, "Alternate environment files")
	flags.StringArrayVar(&req.Compose.Profiles, "profile", nil, "Profiles to enable")
	flags.StringVar(&req.Compose.ProjectDir, "project-directory", "", "Alternate working directory")
	return cmd
}
