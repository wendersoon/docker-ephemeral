// Command docker-ephemeral is a Docker CLI plugin that runs containers and
// Compose projects whose resources are removed when they stop.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/wendersoon/docker-ephemeral/internal/app"
	"github.com/wendersoon/docker-ephemeral/internal/cli"
	"github.com/wendersoon/docker-ephemeral/internal/engine/dockercli"
	"github.com/wendersoon/docker-ephemeral/internal/plugin"
	"github.com/wendersoon/docker-ephemeral/internal/reaper"
	"github.com/wendersoon/docker-ephemeral/internal/session"
	"github.com/wendersoon/docker-ephemeral/internal/version"
)

// dockerBinaryEnv is set by the Docker CLI to the path of the docker binary
// that invoked the plugin.
const dockerBinaryEnv = "DOCKER_CLI_PLUGIN_ORIGINAL_CLI_COMMAND"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	inv := plugin.Parse(args)
	if inv.Metadata {
		if err := plugin.WriteMetadata(os.Stdout, plugin.NewMetadata(version.Version)); err != nil {
			return fail(err)
		}
		return 0
	}

	dir, err := session.DefaultDir()
	if err != nil {
		return fail(err)
	}

	eng := dockercli.New(dockercli.NewProcessExecutor(dockerBinary(), inv.GlobalFlags))
	svc := app.NewService(eng, session.NewFileStore(dir), reaper.New(eng))

	root := cli.NewRootCommand(svc, cli.BuildInfo{Version: version.Version, Commit: version.Commit})
	ctx, stop, err := plugin.TerminationContext(context.Background())
	if err != nil {
		return fail(err)
	}
	defer stop()

	err = cli.Execute(ctx, root, inv.Args)
	var exitErr *cli.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	if err != nil {
		return fail(err)
	}
	return 0
}

func dockerBinary() string {
	if bin := os.Getenv(dockerBinaryEnv); bin != "" {
		return bin
	}
	return "docker"
}

func fail(err error) int {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return 1
}
