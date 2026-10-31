---
hide:
  - navigation
---

# docker ephemeral

**Run containers and Compose projects that leave no trace behind.**

`docker ephemeral` is a Docker CLI plugin. Every workload runs inside a
*session*; when the session ends, every container, network, volume and image
it created is removed. Anything that was already on your machine is left
untouched.

```console
$ docker ephemeral run -it python:3.13 python
Unable to find image 'python:3.13' locally
3.13: Pulling from library/python
...
>>> exit()
Session 3f2a91c04b7e removed: 1 container, 1 image
```

[Get started](getting-started/installation.md){ .md-button .md-button--primary }
[How it works](concepts/how-it-works.md){ .md-button }

## Why

`docker run --rm` and `docker compose down -v --rmi all` cover part of the
problem, but they:

- cannot tell an image pulled **for this workload** from one you already had;
- remove nothing if the process is interrupted or killed;
- leave named volumes created implicitly by `-v name:/path` behind.

`docker ephemeral` records what existed before the workload started and
removes exactly what the workload added.

<div class="grid cards" markdown>

-   :material-console: **Feels like Docker**

    ---

    `run` takes the same arguments as `docker run`; `up` wraps
    `docker compose up`. Global options such as `--context` keep working.

-   :material-shield-check: **Safe by design**

    ---

    Pre-existing images and volumes are never removed. Resources still used
    by other containers are kept and reported.

-   :material-layers-triple: **Compose aware**

    ---

    Databases, caches, custom networks, builds and named volumes of a whole
    stack are torn down together.

-   :material-backup-restore: **Crash tolerant**

    ---

    Sessions are persisted before anything is created, so an interrupted
    cleanup can always be finished with `docker ephemeral gc`.

</div>

## At a glance

```sh
# Throwaway container: removed with its image and volumes on exit
docker ephemeral run --rm -it alpine sh

# Throwaway stack: start in the background...
docker ephemeral up -f compose.yml --name api

# ...and remove everything it created
docker ephemeral down api
```
