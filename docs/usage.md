# Usage

## Containers: `run`

`run` takes the same arguments as `docker run`. The container runs in the
foreground and its exit code is preserved.

```sh
docker ephemeral run --rm -it alpine sh
docker ephemeral run -p 5432:5432 -e POSTGRES_PASSWORD=x postgres:17-alpine
```

When it exits — normally, with ++ctrl+c++ or because `docker` was killed — the
container, its volumes and any image pulled for it are removed.

!!! note
    `run` is foreground only (`-d` is not supported). Use `up` for background
    workloads.

## Compose projects: `up`

```sh
docker ephemeral up                                  # compose.yaml in the current directory
docker ephemeral up -f compose.yml --name api        # custom file and session name
docker ephemeral up web db -- --build --wait         # services, then args for `compose up`
```

`up` starts the project in the background and prints the session ID on stdout,
so scripts can capture it:

```sh
ID=$(docker ephemeral up -- --wait)
trap 'docker ephemeral down "$ID"' EXIT
make test
```

The project runs as `ephemeral-<id>`, so regular Compose commands work with
`-p`:

```sh
docker compose -p ephemeral-<id> exec db psql -U postgres
```

With `--attach`, logs are followed and everything is removed when you press
++ctrl+c++:

```sh
docker ephemeral up --attach
```

## Managing sessions

```console
$ docker ephemeral ls
SESSION ID     NAME   MODE      CONTAINERS    CREATED
50da25697f25   api    compose   4/4 running   3 minutes ago

$ docker ephemeral down api        # by name, ID or ID prefix
Session api removed: 4 containers, 2 networks, 2 volumes, 3 images
```

- `down` without arguments removes the only active session; `down --all` removes every one.
- `gc` removes sessions left behind with no running container (e.g. after a crash).

Docker global options work as usual: `docker --context remote ephemeral up`.
