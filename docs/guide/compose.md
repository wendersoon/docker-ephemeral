# Compose projects

```
docker ephemeral up [flags] [SERVICE...] [-- COMPOSE_UP_ARGS...]
```

`up` runs `docker compose up --detach` under a dedicated project name,
`ephemeral-<session id>`, and prints the session ID.

## Selecting files

Compose files are resolved like `docker compose` does: without `-f`, the
`compose.yaml` (or `docker-compose.yml`) in the current directory is used.

```sh
docker ephemeral up -f compose.yml -f compose.dev.yml --env-file .env.dev --profile debug
```

| Flag | Description |
| --- | --- |
| `-f`, `--file` | Compose configuration file (repeatable) |
| `--env-file` | Alternate environment file (repeatable) |
| `--profile` | Profile to enable (repeatable) |
| `--project-directory` | Alternate working directory |
| `-n`, `--name` | Session name, usable instead of the ID |
| `-a`, `--attach` | Follow logs and remove the session on exit |

Paths are stored as absolute paths, so the session can be managed later from
any directory.

## Passing options to `docker compose up`

Positional arguments select services. Anything after `--` goes straight to
`docker compose up`:

```sh
docker ephemeral up web worker -- --build --wait --scale worker=3
```

## Detached vs attached

=== "Detached (default)"

    ```console
    $ ID=$(docker ephemeral up --name api)
    $ curl localhost:8080
    $ docker ephemeral down "$ID"
    ```

    The session lives until `docker ephemeral down`. Progress goes to stderr
    and only the session ID goes to stdout.

=== "Attached"

    ```console
    $ docker ephemeral up --attach
    db-1     | database system is ready to accept connections
    ^C
    Session 3f2a91c04b7e removed: 2 containers, 1 network, 1 volume, 1 image
    ```

    Logs are followed and the session is removed as soon as you press
    ++ctrl+c++.

## Interacting with the project

The session is a regular Compose project, so every `docker compose` command
works with `-p`:

```sh
docker compose -p ephemeral-3f2a91c04b7e ps
docker compose -p ephemeral-3f2a91c04b7e exec db psql -U postgres
docker compose -p ephemeral-3f2a91c04b7e logs -f web
```

## What gets removed

On `down`, everything Compose created for the project is removed:

- service containers, including stopped and one-off (`run`) containers;
- project networks;
- project volumes and anonymous volumes;
- images built for the project (`<project>-<service>`);
- images pulled for the project, unless they existed before or are still used
  by containers outside the session.

If `up` fails (bad file, failed healthcheck with `--wait`, build error...), the
partial session is removed immediately.
