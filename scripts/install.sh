#!/bin/sh
# Installs the docker-ephemeral CLI plugin for the current user.
#
#   curl -fsSL https://raw.githubusercontent.com/wendersoon/docker-ephemeral/main/scripts/install.sh | sh
#
# Environment:
#   VERSION        release tag to install (default: latest)
#   DOCKER_CONFIG  Docker configuration directory (default: ~/.docker)
set -eu

REPO="wendersoon/docker-ephemeral"
BINARY="docker-ephemeral"
VERSION="${VERSION:-latest}"
PLUGIN_DIR="${DOCKER_CONFIG:-$HOME/.docker}/cli-plugins"

fail() {
  echo "error: $*" >&2
  exit 1
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo amd64 ;;
    aarch64 | arm64) echo arm64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    fail "curl or wget is required"
  fi
}

main() {
  asset="${BINARY}-$(detect_os)-$(detect_arch)"
  if [ "$VERSION" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/${asset}"
  else
    url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
  fi

  tmp="$(mktemp)"
  trap 'rm -f "$tmp"' EXIT

  echo "Downloading ${url}"
  download "$url" "$tmp" || fail "download failed"

  mkdir -p "$PLUGIN_DIR"
  install -m 0755 "$tmp" "${PLUGIN_DIR}/${BINARY}"
  echo "Installed ${PLUGIN_DIR}/${BINARY}"
  docker ephemeral version 2>/dev/null || true
}

main
