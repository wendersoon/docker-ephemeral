# Contributing

Contributions are welcome! Please open an issue to discuss significant changes
before sending a pull request.

## Setup

Requirements: Go 1.23+, Docker with Compose v2, and
[golangci-lint](https://golangci-lint.run) v2.

```sh
git clone https://github.com/wendersoon/docker-ephemeral.git
cd docker-ephemeral
make help
```

| Target | Description |
| --- | --- |
| `make build` | Build the plugin into `bin/` |
| `make install` | Install the plugin for the current user |
| `make test` | Unit tests with race detector and coverage |
| `make test-integration` | End-to-end tests against the local Docker engine |
| `make lint` | Run golangci-lint |
| `make fmt` | Format the code |

Integration tests build the plugin into an isolated `DOCKER_CONFIG`, so they do
not touch your installed copy, and they skip themselves rather than delete an
image you already have.

## Guidelines

- Keep the layers separate: the application depends on the `engine.Engine`
  port, never on the docker CLI directly.
- Cover behavior with unit tests using `enginetest`, and with an integration
  test when it depends on real Docker behavior.
- Run `make lint test` before pushing.

## Commit messages

Commits follow [Conventional Commits](https://www.conventionalcommits.org) and
stay short:

```
feat(cli): add --format flag to ls
fix(reaper): keep volumes used by other containers
docs: document remote contexts
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`, `chore`.

## Releases

Pushing a `v*` tag runs GoReleaser, which builds the binaries for Linux, macOS
and Windows (amd64 and arm64) and publishes a GitHub release with checksums and
a changelog generated from the commit history.

```sh
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

## Documentation

The site is built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/)
and deployed to GitHub Pages on every push to `main`.

```sh
pip install -r docs/requirements.txt
mkdocs serve
```
