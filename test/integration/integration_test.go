//go:build integration

// Package integration exercises the plugin against a real Docker engine.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
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

func TestSharedImage_RemovedByLastSession(t *testing.T) {
	requireDocker(t)
	cfg := dockerConfig(t)
	if mustDocker(t, cfg, "image", "ls", "-q", testImage) != "" {
		t.Skipf("%s already present; refusing to touch it", testImage)
	}

	file := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  hello:\n    image: "+testImage+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	first := mustDocker(t, cfg, "ephemeral", "up", "-f", file)
	second := mustDocker(t, cfg, "ephemeral", "up", "-f", file)

	mustDocker(t, cfg, "ephemeral", "down", first)
	if mustDocker(t, cfg, "image", "ls", "-q", testImage) == "" {
		t.Fatal("image removed while still used by another session")
	}

	mustDocker(t, cfg, "ephemeral", "down", second)
	if out := mustDocker(t, cfg, "image", "ls", "-q", testImage); out != "" {
		t.Errorf("shared image survived the last session: %s", out)
	}
}

func TestRun_CleansUpWhenDockerIsTerminated(t *testing.T) {
	requireDocker(t)
	cfg := dockerConfig(t)

	cmd := exec.Command("docker", "ephemeral", "run", "busybox:1.36",
		"sh", "-c", "trap 'exit 0' TERM; while :; do sleep 1; done")
	cmd.Env = append(os.Environ(), "DOCKER_CONFIG="+cfg)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool { return mustDocker(t, cfg, "ephemeral", "ls", "-q") != "" })
	waitFor(t, func() bool {
		return mustDocker(t, cfg, "container", "ls", "-q", "--filter", "label=io.github.wendersoon.ephemeral.session") != ""
	})
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait() // the exit status depends on how docker reports the interruption
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("plugin did not terminate")
	}

	if out := mustDocker(t, cfg, "ephemeral", "ls", "-q"); out != "" {
		t.Errorf("session survived termination: %s", out)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for range 60 {
		if cond() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func TestDown_RestoresTagTakenFromExistingImage(t *testing.T) {
	requireDocker(t)
	cfg := dockerConfig(t)

	const tag = "docker-ephemeral-test:restore"
	if mustDocker(t, cfg, "image", "ls", "-q", tag) != "" {
		t.Skipf("%s already present; refusing to touch it", tag)
	}
	pulled := mustDocker(t, cfg, "image", "ls", "-q", testImage) == ""
	mustDocker(t, cfg, "pull", "-q", testImage)
	mustDocker(t, cfg, "tag", testImage, tag)
	t.Cleanup(func() {
		_, _ = docker(t, cfg, "image", "rm", tag)
		if pulled {
			_, _ = docker(t, cfg, "image", "rm", testImage)
		}
	})
	before := mustDocker(t, cfg, "image", "inspect", "--format", "{{.ID}}", tag)

	// The session builds a different image under the same tag.
	file := filepath.Join(t.TempDir(), "compose.yml")
	compose := "services:\n  app:\n    image: " + tag + "\n    build:\n      context: .\n" +
		"      dockerfile_inline: \"FROM " + testImage + "\\nLABEL session=1\"\n"
	if err := os.WriteFile(file, []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}

	id := mustDocker(t, cfg, "ephemeral", "up", "-f", file, "--", "--build")
	t.Cleanup(func() { _, _ = docker(t, cfg, "ephemeral", "down", id) })
	if mustDocker(t, cfg, "image", "inspect", "--format", "{{.ID}}", tag) == before {
		t.Fatal("test setup: the session did not take the tag over")
	}

	mustDocker(t, cfg, "ephemeral", "down", id)
	if after := mustDocker(t, cfg, "image", "inspect", "--format", "{{.ID}}", tag); after != before {
		t.Errorf("tag points to %s, want the original image %s", after, before)
	}
}
