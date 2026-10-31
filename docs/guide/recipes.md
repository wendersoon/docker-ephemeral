# Recipes

## Database for a test run

```sh
docker ephemeral run -p 5432:5432 -e POSTGRES_PASSWORD=test postgres:17-alpine &
# ...run your tests against localhost:5432...
kill %1   # stops postgres and removes the container, volume and image
```

Or with Compose and healthchecks:

```yaml title="compose.test.yml"
services:
  db:
    image: postgres:17-alpine
    environment: { POSTGRES_PASSWORD: test }
    ports: ["5432:5432"]
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 2s
      retries: 15
```

```sh
ID=$(docker ephemeral up -f compose.test.yml -- --wait)
trap 'docker ephemeral down "$ID"' EXIT
go test ./...
```

## CI jobs

```yaml title=".github/workflows/test.yml"
- name: Install docker ephemeral
  run: curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | sh

- name: Integration tests
  run: |
    ID=$(docker ephemeral up -f compose.test.yml -- --wait)
    make test-integration
    docker ephemeral down "$ID"
```

On self-hosted runners this keeps disk usage stable across jobs.

## Trying an image

```sh
docker ephemeral run -it --rm node:22 node
docker ephemeral run -it --rm golang:1.25 bash
```

Nothing is left behind, including the multi-hundred-megabyte image.

## Full stack demo

```yaml title="compose.yml"
services:
  db:
    image: postgres:17-alpine
    environment: { POSTGRES_PASSWORD: secret }
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck: { test: ["CMD", "pg_isready", "-U", "postgres"], interval: 2s, retries: 15 }
  cache:
    image: redis:7-alpine
    volumes: [cache:/data]
  adminer:
    image: adminer
    depends_on: { db: { condition: service_healthy } }
    ports: ["8080:8080"]
volumes: { pgdata: {}, cache: {} }
```

```sh
docker ephemeral up --attach -- --wait
```

Open <http://localhost:8080>, play around, then press ++ctrl+c++.

## Remote engines

Docker global options are forwarded to every call:

```sh
docker --context staging ephemeral up -f compose.yml
docker --context staging ephemeral down --all
```
