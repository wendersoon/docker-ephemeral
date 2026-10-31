# Contributing

Thanks for your interest! Bug reports, ideas and pull requests are welcome.

## Reporting

- **Bugs and ideas:** open an [issue](https://github.com/wendersoon/docker-ephemeral/issues/new/choose).
- **Security problems** (e.g. something deleted that shouldn't be): see [SECURITY.md](SECURITY.md).

For larger changes, open an issue first so we can agree on the approach.

## Development

Requirements: Go 1.26+, Docker with Compose v2 and [golangci-lint](https://golangci-lint.run) v2.

```sh
make build             # build into bin/
make install           # install the plugin for your user
make test              # unit tests
make test-integration  # end-to-end tests against the local Docker engine
make lint              # linters
```

Integration tests use an isolated `DOCKER_CONFIG` and never touch images you
already have.

## Pull requests

- Keep changes focused and covered by tests: unit tests with the in-memory
  engine (`internal/engine/enginetest`), plus an integration test when the
  behavior depends on real Docker.
- Run `make lint test` before pushing; CI must pass.
- Use [Conventional Commits](https://www.conventionalcommits.org), short and in
  English: `fix(reaper): keep volumes used by other containers`.

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE) and to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
