# Quick start

This tour takes about two minutes and shows the two ways of running a workload.

## 1. A throwaway container

```console
$ docker ephemeral run --rm -it alpine sh
/ # echo hello > /tmp/file
/ # exit
Session b5620067a026 removed: 1 container, 1 image
```

The `alpine` image was not on the machine, so it was pulled for the session and
removed with it. Run it again with an image you already have and it is kept:

```console
$ docker pull busybox
$ docker ephemeral run busybox echo hi
hi
Session 7e01c2d3a9f4 removed: 1 container
```

## 2. A throwaway stack

Create a `compose.yml`:

```yaml title="compose.yml"
services:
  db:
    image: postgres:17-alpine
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
  cache:
    image: redis:7-alpine

volumes:
  pgdata:
```

Start it as a session. The session ID is printed on stdout, so it can be
captured by scripts:

```console
$ docker ephemeral up --name demo -- --wait
 ✔ Container ephemeral-50da25697f25-db-1     Healthy
 ✔ Container ephemeral-50da25697f25-cache-1  Healthy
50da25697f25
Session demo is up. Remove it with: docker ephemeral down 50da25697f25
```

Work with it as with any Compose project; the project name is
`ephemeral-<session id>`:

```sh
docker compose -p ephemeral-50da25697f25 exec db psql -U postgres
```

When you are done:

```console
$ docker ephemeral down demo
Session demo removed: 2 containers, 1 network, 1 volume, 2 images
```

!!! tip "Prefer one terminal?"
    `docker ephemeral up --attach` follows the logs and removes the session as
    soon as you press ++ctrl+c++.

## Next steps

- [Running containers](../guide/run.md)
- [Compose projects](../guide/compose.md)
- [How it works](../concepts/how-it-works.md)
