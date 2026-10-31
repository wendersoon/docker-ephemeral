package dockercli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

// fakeExecutor answers commands by their joined arguments.
type fakeExecutor struct {
	responses map[string]Result
	errors    map[string]error
	calls     []Command
}

func (f *fakeExecutor) Execute(_ context.Context, c Command) (Result, error) {
	f.calls = append(f.calls, c)
	key := strings.Join(c.Args, " ")
	return f.responses[key], f.errors[key]
}

func TestContainers_InspectsMatches(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]Result{
		"container ls --all --quiet --no-trunc --filter label=k=v": {Stdout: []byte("c1\n")},
		"container inspect c1": {Stdout: []byte(`[{
			"Id": "c1", "Name": "/web", "Image": "sha256:img",
			"State": {"Running": true}, "Config": {"OpenStdin": true},
			"Mounts": [{"Type": "volume", "Name": "data"}, {"Type": "bind", "Name": ""}]
		}]`)},
	}}

	got, err := New(exec).Containers(context.Background(), session.Label{Key: "k", Value: "v"})
	if err != nil {
		t.Fatal(err)
	}
	want := []engine.Container{{
		ID: "c1", Name: "web", ImageID: "sha256:img", Running: true, Interactive: true, Volumes: []string{"data"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Containers() = %+v, want %+v", got, want)
	}
}

func TestContainers_NoMatchSkipsInspect(t *testing.T) {
	exec := &fakeExecutor{}
	got, err := New(exec).Containers(context.Background(), session.Label{Key: "k", Value: "v"})
	if err != nil || got != nil || len(exec.calls) != 1 {
		t.Errorf("got %v, %v after %d calls", got, err, len(exec.calls))
	}
}

func TestCreateContainer_LabelsAndReturnsID(t *testing.T) {
	key := "container create --label k=v --rm alpine true"
	exec := &fakeExecutor{responses: map[string]Result{key: {Stdout: []byte("abc\n")}}}

	id, err := New(exec).CreateContainer(context.Background(), session.Label{Key: "k", Value: "v"}, []string{"--rm", "alpine", "true"})
	if err != nil || id != "abc" {
		t.Fatalf("got %q, %v", id, err)
	}
	if !exec.calls[0].ShowProgress {
		t.Error("expected pull progress to be shown")
	}
}

func TestStartAttached_Interactive(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]Result{
		"container start --attach --interactive c1": {ExitCode: 3},
	}}

	code, err := New(exec).StartAttached(context.Background(), engine.Container{ID: "c1", Interactive: true})
	if err != nil || code != 3 {
		t.Errorf("got %d, %v", code, err)
	}
	if !exec.calls[0].Attached {
		t.Error("expected attached command")
	}
}

func TestComposeUp_BuildsProjectArgs(t *testing.T) {
	exec := &fakeExecutor{}
	project := session.Compose{
		Project: "ephemeral-x", ProjectDir: "/app",
		Files: []string{"a.yml", "b.yml"}, EnvFiles: []string{".env"}, Profiles: []string{"dev"},
	}

	if _, err := New(exec).ComposeUp(context.Background(), project, []string{"--build"}); err != nil {
		t.Fatal(err)
	}
	want := "compose --project-name ephemeral-x --project-directory /app --file a.yml --file b.yml " +
		"--env-file .env --profile dev up --detach --build"
	if got := strings.Join(exec.calls[0].Args, " "); got != want {
		t.Errorf("args = %q\nwant  %q", got, want)
	}
	if !exec.calls[0].StdoutToStderr {
		t.Error("compose up output must not pollute stdout")
	}
}

func TestImageTags(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]Result{
		"image inspect --format {{json .RepoTags}} img": {Stdout: []byte(`["a:1","b:2"]` + "\n")},
	}}
	got, err := New(exec).ImageTags(context.Background(), "img")
	if err != nil || !reflect.DeepEqual(got, []string{"a:1", "b:2"}) {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestClassify(t *testing.T) {
	tests := map[string]error{
		"Error response from daemon: No such volume: x":                                  engine.ErrNotFound,
		"Error response from daemon: remove x: volume is in use - [abc]":                 engine.ErrInUse,
		"Error response from daemon: error while removing network: has active endpoints": engine.ErrInUse,
		"something else": nil,
	}
	for msg, want := range tests {
		err := &CommandError{Stderr: msg, kind: classify(msg)}
		if want != nil && !errors.Is(err, want) {
			t.Errorf("classify(%q) does not match %v", msg, want)
		}
		if want == nil && err.Unwrap() != nil {
			t.Errorf("classify(%q) = %v, want nil", msg, err.Unwrap())
		}
	}
}
