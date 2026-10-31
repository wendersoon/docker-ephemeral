# Labels and state

## Labels

| Label | Set on | Value |
| --- | --- | --- |
| `io.github.wendersoon.ephemeral.session` | Containers created by `run` | Session ID |
| `com.docker.compose.project` | Every resource of an `up` session (set by Compose) | `ephemeral-<session id>` |

You can use them with the regular Docker commands:

```sh
docker ps --filter label=io.github.wendersoon.ephemeral.session
docker network ls --filter label=com.docker.compose.project=ephemeral-50da25697f25
```

## State directory

Sessions are stored as JSON files in:

```
${DOCKER_CONFIG:-~/.docker}/ephemeral/sessions/<session id>.json
```

Files are written atomically and removed once the session has been torn down.

```json title="50da25697f25.json"
{
  "id": "50da25697f25",
  "name": "dbstack",
  "mode": "compose",
  "createdAt": "2026-10-31T21:05:12.418Z",
  "selector": {
    "key": "com.docker.compose.project",
    "value": "ephemeral-50da25697f25"
  },
  "compose": {
    "project": "ephemeral-50da25697f25",
    "files": ["/home/me/stack/compose.yml"]
  },
  "baseline": {
    "images": ["sha256:ab20e934b27b..."],
    "volumes": ["my-data"]
  },
  "owned": {
    "images": ["sha256:7e49ed81b42b...", "sha256:ab20e934b27b..."],
    "volumes": ["ephemeral-50da25697f25_pgdata"]
  }
}
```

| Field | Meaning |
| --- | --- |
| `baseline` | Resources found before the session started; never removed |
| `owned` | Resources observed in use by the session's containers |
| `selector` | Label matching the session's containers, networks and volumes |

Only `owned` resources that are not in `baseline` are removed.

!!! warning
    The state format is not a stable API before version 1.0.
