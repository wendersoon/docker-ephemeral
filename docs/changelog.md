# Changelog

All notable changes are documented here. The project follows
[Semantic Versioning](https://semver.org).

## Unreleased

### Fixes

- Stop the workload and clean up when the Docker CLI is terminated
  (`kill`, `timeout`, CI cancellation). Newer Docker CLIs notify plugins through
  a socket instead of forwarding signals.
- Images shared by concurrent sessions are now removed by the last session
  using them instead of being left behind.

## 0.1.0 — 2026-10-31

First release.

- `run`: run a container in the foreground and remove the container, its
  volumes and pulled images on exit, preserving the exit code.
- `up`: start a Compose project as a session, detached or attached.
- `down`, `ls`, `gc`: manage and recover sessions.
- Install script and binaries for Linux, macOS and Windows.
