# How it works

1. **Snapshot.** Before anything is created, the images and volumes already on
   the host are recorded. They are never removed.
2. **Track.** Containers are labeled with the session (`run`) or run under the
   Compose project `ephemeral-<id>` (`up`). The images and volumes they use are
   recorded right away.
3. **Tear down.** Containers, networks, volumes and images of the session are
   removed in that order.

Sessions are saved in `~/.docker/ephemeral/` before the first resource is
created, so an interrupted cleanup can always be finished with
`docker ephemeral down` or `gc`.

## What is kept

- Images and volumes that existed before the session.
- Anything still used by a container outside the session — it is reported:

    ```console
    Session A removed: 1 container, 1 network
      kept image 3d6cab9c2941: in use outside the session
    ```

- Bind-mounted host directories.

When two sessions share a newly pulled image, the last one to end removes it.

## What is removed

Containers, networks, volumes (**with their data**) and images pulled or built
for the session. Use a bind mount or a pre-existing volume for data you want to
keep.

## Limitations

- BuildKit cache is not pruned (`docker builder prune`).
- Images pulled by Compose for services that never got a container (e.g. `up`
  interrupted mid-pull) are not tracked.
