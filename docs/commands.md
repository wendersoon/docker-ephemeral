# Commands

| Command | Description |
| --- | --- |
| `run [OPTIONS] IMAGE [COMMAND]` | Run a container in the foreground; same options as `docker run` except `-d` |
| `up [flags] [SERVICE...] [-- ARGS]` | Start a Compose project in the background and print the session ID |
| `down [SESSION...]` | Remove sessions by name, ID or ID prefix (`--all` for every session) |
| `ls` | List active sessions (`-q` for IDs only) |
| `gc` | Remove sessions with no running container |
| `version` | Show the version |

## `up` flags

| Flag | Description |
| --- | --- |
| `-f`, `--file` | Compose file (repeatable) |
| `-n`, `--name` | Session name |
| `-a`, `--attach` | Follow logs and remove the session on exit |
| `--env-file` | Environment file (repeatable) |
| `--profile` | Profile to enable (repeatable) |
| `--project-directory` | Alternate working directory |

Arguments after `--` are forwarded to `docker compose up`.

## Exit codes

`run` exits with the container's exit code. Other commands exit with `0` on
success and `1` on error.
