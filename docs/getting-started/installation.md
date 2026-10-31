# Installation

`docker ephemeral` is a single static binary installed as a
[Docker CLI plugin](https://docs.docker.com/engine/extend/cli_plugins/).

## Requirements

- Docker Engine or Docker Desktop with the `docker` CLI.
- Docker Compose v2 (the `docker compose` plugin) for the `up` command.

## Install script

=== "Latest release"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | sh
    ```

=== "Specific version"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | VERSION=v0.1.0 sh
    ```

The script detects your OS (Linux or macOS) and architecture (amd64 or arm64)
and installs the plugin into `~/.docker/cli-plugins`, or
`$DOCKER_CONFIG/cli-plugins` when `DOCKER_CONFIG` is set.

## Manual download

Download the binary for your platform from the
[releases page](https://github.com/wendersoon/docker-ephemeral/releases), then:

=== "Linux / macOS"

    ```sh
    mkdir -p ~/.docker/cli-plugins
    mv docker-ephemeral-<os>-<arch> ~/.docker/cli-plugins/docker-ephemeral
    chmod +x ~/.docker/cli-plugins/docker-ephemeral
    ```

=== "Windows (PowerShell)"

    ```powershell
    New-Item -ItemType Directory -Force "$env:USERPROFILE\.docker\cli-plugins"
    Move-Item docker-ephemeral-windows-amd64.exe "$env:USERPROFILE\.docker\cli-plugins\docker-ephemeral.exe"
    ```

Every release ships a `checksums.txt` file to verify the download:

```sh
sha256sum --ignore-missing -c checksums.txt
```

## From source

Requires Go 1.23 or later.

```sh
git clone https://github.com/wendersoon/docker-ephemeral.git
cd docker-ephemeral
make install
```

## Verify

```console
$ docker ephemeral version
docker ephemeral 0.1.0 (commit 71d031e)
```

The plugin is also listed by `docker info` under *Client → Plugins*.

## Uninstall

```sh
docker ephemeral down --all   # optional: remove active sessions first
rm ~/.docker/cli-plugins/docker-ephemeral
rm -rf ~/.docker/ephemeral
```
