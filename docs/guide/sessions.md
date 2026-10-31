# Managing sessions

## Listing

```console
$ docker ephemeral ls
SESSION ID     NAME      MODE      CONTAINERS    CREATED
50da25697f25   dbstack   compose   4/4 running   3 minutes ago
d62c5c05b8b6             run       1/1 running   less than a minute ago
```

`-q` prints only the IDs, which is handy for scripting:

```sh
for id in $(docker ephemeral ls -q); do echo "$id"; done
```

## Referencing a session

Every command that takes a session accepts:

- its **name** (`--name` on `up`);
- its full **ID**;
- any unambiguous **ID prefix** (`50da`).

## Removing

```sh
docker ephemeral down                # the only active session
docker ephemeral down dbstack 7e01   # specific sessions
docker ephemeral down --all          # every session
```

Without arguments, `down` refuses to guess when more than one session exists.

## Recovering from crashes

Sessions are written to disk before any resource is created. If
`docker ephemeral` is killed, the machine reboots or the cleanup fails, the
session stays listed and can be removed later.

`gc` removes every session that has no running container, keeping the ones
that are still in use:

```console
$ docker ephemeral gc
Session d62c5c05b8b6 removed: 1 container, 1 volume, 1 image
```

!!! tip
    `docker ephemeral gc` is safe to run from a cron job or a shell login
    script.

## Partial removal

If some resource cannot be removed because it is still in use outside the
session, it is kept and reported, and the session is still considered done:

```console
$ docker ephemeral down A
Session A removed: 1 container, 1 network
  kept image 3d6cab9c2941: in use outside the session
```

Any other failure (daemon unreachable, permission denied...) keeps the session
so that `down` can be retried.
