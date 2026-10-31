---
hide:
  - navigation
  - toc
---

# docker ephemeral

**Use Docker without piling up garbage.**

Run a container or a Compose stack, and when you take it down, everything it
created — containers, images, volumes and networks — is deleted with it.
What you already had on your machine stays untouched.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | sh
```

## Use

```sh
# Run a container; when it exits, nothing is left behind
docker ephemeral run -it alpine sh

# Start a Compose stack...
docker ephemeral up --name app

# ...and take it down with everything it created
docker ephemeral down app
```

`run` accepts the same options as `docker run`, and `up` works with your
existing `compose.yml`.

| Command | What it does |
| --- | --- |
| `run` | Runs a container and deletes everything it created when it exits |
| `up` | Starts a Compose stack in the background |
| `down` | Takes a stack down and deletes everything it created |
| `ls` | Lists what is running |
| `gc` | Cleans up anything left by an interrupted run |

!!! warning "What gets deleted"
    Only what the session brought: images and volumes you already had are never
    deleted, and a tag the session replaced is given back to your image. Data
    in the session's volumes is deleted with it. One edge case remains: an
    image pulled by another process *while* a session is starting, and used by
    it, is treated as the session's. Provided as is, under the MIT license.
