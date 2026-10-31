# Running containers

```
docker ephemeral run [OPTIONS] IMAGE [COMMAND] [ARG...]
```

`run` accepts exactly the same arguments as
[`docker run`](https://docs.docker.com/reference/cli/docker/container/run/).
Everything after `run` is forwarded unchanged.

## Lifecycle

1. A session is created and the images and volumes already present are recorded.
2. The container is created (pulling the image if needed) and labeled with the
   session ID.
3. The container is started in the foreground, attached to your terminal.
4. When it exits, the container, its anonymous and named volumes and any image
   pulled for it are removed.

The exit code of the container becomes the exit code of `docker ephemeral`, so
it can be used in scripts and CI:

```console
$ docker ephemeral run alpine sh -c 'exit 4'; echo $?
Session b5620067a026 removed: 1 container, 1 image
4
```

## Interrupting

Pressing ++ctrl+c++ stops the container like `docker run` would, and the
session is still cleaned up afterwards. The same happens when `docker` is
terminated with `SIGTERM` or `SIGHUP` (for example by `kill`, `timeout` or a CI
runner cancelling a job): the container is stopped and the session removed.

!!! note
    When the session is interrupted this way, the exit code reflects the
    interruption rather than the container's own exit code.

```console
$ docker ephemeral run -e POSTGRES_PASSWORD=x -v pg:/var/lib/postgresql/data postgres:16-alpine
...
LOG:  database system is ready to accept connections
^C
Session d62c5c05b8b6 removed: 1 container, 1 volume, 1 image
```

## Volumes

| Mount | Behavior |
| --- | --- |
| Anonymous volume (`-v /data` or `VOLUME` in the image) | Removed |
| Named volume created by the session (`-v name:/data`) | Removed |
| Named volume that already existed | Kept |
| Bind mount (`-v ./dir:/data`) | Never touched |

!!! warning
    Data written to session volumes is gone once the session ends. Use a bind
    mount or a pre-existing volume for anything you want to keep.

## Interactive use

TTY and stdin options work as usual:

```sh
docker ephemeral run -it ubuntu bash
docker ephemeral run -i alpine wc -l < file.txt
```

## Background containers

`run` always runs in the foreground (`-d` is not supported). For background
workloads, describe them in a Compose file and use [`up`](compose.md).
