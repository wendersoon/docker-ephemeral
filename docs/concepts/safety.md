# Safety guarantees

Deleting things automatically is only useful if it never deletes the wrong
thing. These are the rules `docker ephemeral` follows.

## What is never removed

!!! success "Guaranteed"
    - Images and volumes that existed before the session started.
    - Containers, networks and volumes that do not carry the session selector.
    - Bind-mounted host directories.
    - Images and volumes still used by a container outside the session.

Removals never use `--force` on images or volumes, so the engine itself refuses
to delete anything that is in use.

## What is removed

!!! danger "Removed when the session ends"
    - Containers created by the session, running or stopped.
    - Networks created by the session.
    - Volumes created by the session, named or anonymous, **including their data**.
    - Images pulled or built for the session.

## Shared resources

When two sessions use an image that neither of them found on the host, the
first session to end keeps it and reports it:

```console
$ docker ephemeral down A
Session A removed: 1 container, 1 network
  kept image 3d6cab9c2941: in use outside the session
$ docker ephemeral down B
Session B removed: 1 container, 1 network, 1 image
```

## Failure handling

- A resource that is already gone is ignored.
- A resource in use outside the session is kept and reported.
- Any other error (daemon unreachable, permission denied...) keeps the session
  on disk so the teardown can be retried with `down` or `gc`.

## Known gaps

- **Concurrent pulls outside sessions.** An image pulled by something else
  *while* a session is running, and then used by that session, is considered
  owned by the session.
- **BuildKit cache** is not pruned. Use `docker builder prune` if needed.
- **Images pulled by Compose for services that never got a container** (for
  example when `up` is interrupted mid-pull) are not tracked.
