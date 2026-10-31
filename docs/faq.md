# FAQ

??? question "How is this different from `docker run --rm`?"
    `--rm` removes the container and its anonymous volumes. It keeps the image
    that was pulled for it and any named volume created by `-v name:/path`, and
    it does nothing if the client is killed. `docker ephemeral run` removes
    those too and can recover from interruptions.

??? question "How is this different from `docker compose down -v --rmi all`?"
    `--rmi all` removes every image used by the project, **including images you
    already had**, such as a `postgres` image other projects rely on.
    `docker ephemeral` only removes images that did not exist before the session.

??? question "Will it delete my existing images or volumes?"
    No. Everything present before the session starts is recorded and never
    removed, and nothing in use by other containers is deleted. See
    [Safety guarantees](concepts/safety.md).

??? question "Where does my data go?"
    Data in session volumes is deleted with the session. Use a bind mount
    (`-v ./data:/data`) or a volume that already exists to keep it.

??? question "Can I run containers in the background?"
    Use `up` with a Compose file. `run` is foreground only.

??? question "Does it work with Docker Desktop, remote contexts or Podman?"
    It works with anything the `docker` CLI talks to, including Docker Desktop
    and remote contexts (`docker --context remote ephemeral ...`). Podman's
    Docker-compatible CLI is not tested.

??? question "The plugin crashed. Is something left behind?"
    Possibly, but it is tracked. Run `docker ephemeral ls` to see it and
    `docker ephemeral gc` (or `down`) to remove it.

??? question "Is it stable?"
    It is pre-1.0: the behavior is covered by unit and integration tests, but
    flags, labels and the state format may still change between minor versions.
