package cli

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/app"
	"github.com/wendersoon/docker-ephemeral/internal/reaper"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

type fakeService struct {
	runArgs  []string
	runCode  int
	upReq    app.UpRequest
	downRefs []string
	downAll  bool
	statuses []app.Status
}

var testSession = &session.Session{ID: "a1b2c3d4e5f6", Name: "api", Mode: session.ModeCompose, CreatedAt: time.Now()}

func (f *fakeService) Run(_ context.Context, args []string) (app.RunResult, error) {
	f.runArgs = args
	return app.RunResult{Teardown: app.Teardown{Session: testSession}, ExitCode: f.runCode}, nil
}

func (f *fakeService) Up(_ context.Context, req app.UpRequest) (app.UpResult, error) {
	f.upReq = req
	return app.UpResult{Teardown: app.Teardown{Session: testSession}}, nil
}

func (f *fakeService) Down(_ context.Context, refs []string, all bool) ([]app.Teardown, error) {
	f.downRefs, f.downAll = refs, all
	return []app.Teardown{{Session: testSession, Report: reaper.Report{Removed: map[reaper.Kind][]string{
		reaper.KindContainer: {"c1"},
	}}}}, nil
}

func (f *fakeService) List(context.Context) ([]app.Status, error) { return f.statuses, nil }

func (f *fakeService) GC(context.Context) ([]app.Teardown, error) { return nil, nil }

func execute(svc Service, args ...string) (stdout, stderr string, err error) {
	var out, errOut bytes.Buffer
	root := NewRootCommand(svc, BuildInfo{Version: "1.0.0", Commit: "abc"})
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestRun_ForwardsArgsAndExitCode(t *testing.T) {
	svc := &fakeService{runCode: 3}
	_, _, err := execute(svc, "run", "--rm", "-it", "alpine", "sh")

	if !slices.Equal(svc.runArgs, []string{"--rm", "-it", "alpine", "sh"}) {
		t.Errorf("args = %q", svc.runArgs)
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Errorf("err = %v", err)
	}
}

func TestRun_RequiresImage(t *testing.T) {
	if _, _, err := execute(&fakeService{}, "run"); err == nil {
		t.Error("expected error")
	}
}

func TestUp_ParsesFlagsAndPassthrough(t *testing.T) {
	svc := &fakeService{}
	stdout, _, err := execute(svc, "up", "-f", "a.yml", "-f", "b.yml", "--name", "api", "web", "--", "--build")
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(svc.upReq.Compose.Files, []string{"a.yml", "b.yml"}) || svc.upReq.Name != "api" {
		t.Errorf("request = %+v", svc.upReq)
	}
	if !slices.Equal(svc.upReq.Args, []string{"web", "--build"}) {
		t.Errorf("args = %q", svc.upReq.Args)
	}
	if strings.TrimSpace(stdout) != testSession.ID {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestDown(t *testing.T) {
	svc := &fakeService{}
	_, stderr, err := execute(svc, "down", "api")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(svc.downRefs, []string{"api"}) || svc.downAll {
		t.Errorf("refs = %q all = %v", svc.downRefs, svc.downAll)
	}
	if !strings.Contains(stderr, "Session api removed: 1 container") {
		t.Errorf("stderr = %q", stderr)
	}

	if _, _, err := execute(svc, "down", "--all", "api"); err == nil {
		t.Error("expected --all with references to fail")
	}
}

func TestList(t *testing.T) {
	svc := &fakeService{statuses: []app.Status{{Session: testSession, Containers: 2, Running: 1}}}

	stdout, _, err := execute(svc, "ls")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SESSION ID", "a1b2c3d4e5f6", "api", "compose", "1/2 running"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q:\n%s", want, stdout)
		}
	}

	stdout, _, _ = execute(svc, "ls", "-q")
	if strings.TrimSpace(stdout) != testSession.ID {
		t.Errorf("quiet output = %q", stdout)
	}
}

func TestHumanizeSince(t *testing.T) {
	now := time.Now()
	tests := map[time.Duration]string{
		10 * time.Second: "less than a minute ago",
		time.Minute:      "1 minute ago",
		3 * time.Hour:    "3 hours ago",
		50 * time.Hour:   "2 days ago",
	}
	for d, want := range tests {
		if got := humanizeSince(now.Add(-d), now); got != want {
			t.Errorf("humanizeSince(%s) = %q, want %q", d, got, want)
		}
	}
}
