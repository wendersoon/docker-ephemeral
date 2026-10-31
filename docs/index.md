# docker ephemeral

**Run containers and Compose projects that leave no trace behind.**

A Docker CLI plugin: when a workload stops, every container, network, volume
and image it created is removed. Anything that was already on your machine is
left untouched.

```console
$ docker ephemeral run -it python:3.13 python
Unable to find image 'python:3.13' locally
...
>>> exit()
Session 3f2a91c04b7e removed: 1 container, 1 image
```

## Why not `--rm` or `compose down`?

- `docker run --rm` keeps the pulled image and named volumes, and does nothing
  if the client is killed.
- `docker compose down -v --rmi all` also removes images **you already had**.

`docker ephemeral` records what existed before the workload started and removes
only what the workload added.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | sh
docker ephemeral version
```

The binary goes to `~/.docker/cli-plugins`. Binaries for Linux, macOS and
Windows are also on the [releases page](https://github.com/wendersoon/docker-ephemeral/releases),
and `make install` builds from source.

## In 30 seconds

```sh
# A container: removed with its image and volumes when it exits
docker ephemeral run --rm -it alpine sh

# A Compose stack: start it...
docker ephemeral up --name demo

# ...and remove everything it created
docker ephemeral down demo
```

Next: [Usage](usage.md).
