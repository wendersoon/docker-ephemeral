//go:build integration

// Package integration exercises the plugin against a real Docker engine.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testImage = "hello-world:latest"

// dockerConfig builds the plugin into an isolated DOCKER_CONFIG directory.
func dockerConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "cli-plugins", "docker-ephemeral")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/docker-ephemeral")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, out)
	}
	return dir
}

func docker(t *testing.T, cfg string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), "DOCKER_CONFIG="+cfg)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func mustDocker(t *testing.T, cfg string, args ...string) string {
	t.Helper()
	out, err := docker(t, cfg, args...)
	if err != nil {
		t.Fatalf("docker %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker engine not available")
	}
}

func TestRun_RemovesPulledImageAndKeepsExitCode(t *testing.T) {
	requireDocker(t)
	cfg := dockerConfig(t)
	if mustDocker(t, cfg, "image", "ls", "-q", testImage) != "" {
		t.Skipf("%s already present; refusing to touch it", testImage)
	}

	if _, err := docker(t, cfg, "ephemeral", "run", testImage); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out := mustDocker(t, cfg, "image", "ls", "-q", testImage); out != "" {
		t.Errorf("pulled image survived: %s", out)
	}

	_, err := docker(t, cfg, "ephemeral", "run", "--entrypoint", "/nonexistent", testImage)
	if err == nil {
		t.Error("expected a failing container to propagate its exit code")
	}
}

func TestUpDown_RemovesComposeProject(t *testing.T) {
	requireDocker(t)
	cfg := dockerConfig(t)

	dir := t.TempDir()
	compose := `services:
  app:
    build:
      context: .
      dockerfile_inline: "FROM hello-world\nCOPY compose.yml /"
    volumes: [data:/data]
volumes:
  data:
`
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}

	id := mustDocker(t, cfg, "ephemeral", "up", "-f", file)
	if id == "" {
		t.Fatal("up did not print a session id")
	}
	if !strings.Contains(mustDocker(t, cfg, "ephemeral", "ls", "-q"), id) {
		t.Fatal("session not listed")
	}

	mustDocker(t, cfg, "ephemeral", "down", id)

	filter := "label=com.docker.compose.project=ephemeral-" + id
	for _, args := range [][]string{
		{"container", "ls", "-aq", "--filter", filter},
		{"network", "ls", "-q", "--filter", filter},
		{"volume", "ls", "-q", "--filter", filter},
		{"image", "ls", "-q", "ephemeral-" + id + "-app"},
	} {
		if out := mustDocker(t, cfg, args...); out != "" {
			t.Errorf("docker %s left %q", strings.Join(args, " "), out)
		}
	}
	if out := mustDocker(t, cfg, "ephemeral", "ls", "-q"); out != "" {
		t.Errorf("sessions left: %s", out)
	}
}
