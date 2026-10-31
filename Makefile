BINARY     := docker-ephemeral
MODULE     := github.com/wendersoon/docker-ephemeral
PLUGIN_DIR ?= $(or $(DOCKER_CONFIG),$(HOME)/.docker)/cli-plugins
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS    := -s -w -X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.Commit=$(COMMIT)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the plugin into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)

.PHONY: install
install: build ## Install the plugin for the current user
	mkdir -p $(PLUGIN_DIR)
	install -m 0755 bin/$(BINARY) $(PLUGIN_DIR)/$(BINARY)

.PHONY: uninstall
uninstall: ## Remove the installed plugin
	rm -f $(PLUGIN_DIR)/$(BINARY)

.PHONY: test
test: ## Run unit tests
	go test -race -coverprofile=coverage.out ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	go test -tags integration -count=1 ./test/...

.PHONY: lint
lint: ## Run linters
	golangci-lint run

.PHONY: vuln
vuln: ## Check for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: fmt
fmt: ## Format the code
	golangci-lint fmt

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin dist coverage.out
