# Commands

All commands are available as `docker ephemeral <command>`. Docker global
options placed before `ephemeral` (such as `--context`, `--host` or
`--log-level`) are forwarded to every `docker` call.

Informational messages go to **stderr**; machine-readable output (session IDs,
`ls`) goes to **stdout**.

## `run`

```
docker ephemeral run [OPTIONS] IMAGE [COMMAND] [ARG...]
```

Run a container in the foreground and remove everything it created when it
exits. Accepts every option of
[`docker run`](https://docs.docker.com/reference/cli/docker/container/run/)
except `-d/--detach`.

**Exit code:** the exit code of the container.

## `up`

```
docker ephemeral up [flags] [SERVICE...] [-- COMPOSE_UP_ARGS...]
```

Start a Compose project as an ephemeral session and print the session ID on
stdout.

| Flag | Default | Description |
| --- | --- | --- |
| `-a`, `--attach` | `false` | Follow logs and remove the session on exit |
| `--env-file` | | Alternate environment file (repeatable) |
| `-f`, `--file` | | Compose configuration file (repeatable) |
| `-n`, `--name` | | Session name |
| `--profile` | | Profile to enable (repeatable) |
| `--project-directory` | | Alternate working directory |

Arguments after `--` are forwarded to `docker compose up`.

## `down`

```
docker ephemeral down [flags] [SESSION...]
```

Remove sessions and every resource they created. Sessions are referenced by
name, ID or unambiguous ID prefix. Without arguments, removes the only active
session.

| Flag | Description |
| --- | --- |
| `--all` | Remove every session |

## `ls`

```
docker ephemeral ls [flags]
```

List active sessions. Alias: `list`.

| Flag | Description |
| --- | --- |
| `-q`, `--quiet` | Only display session IDs |

## `gc`

```
docker ephemeral gc
```

Remove sessions that have no running container, typically left behind when the
plugin was killed before it could clean up.

## `version`

```
docker ephemeral version
```

Print the plugin version and commit.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Error in `docker ephemeral` or in the workload startup |
| other | Exit code of the container (`run`) |
