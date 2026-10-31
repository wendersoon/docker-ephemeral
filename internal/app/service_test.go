package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/engine/enginetest"
	"github.com/wendersoon/docker-ephemeral/internal/reaper"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

type memStore struct{ sessions []*session.Session }

func (m *memStore) Save(s *session.Session) error {
	if err := m.Delete(s.ID); err != nil {
		return err
	}
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *memStore) List() ([]*session.Session, error) { return slices.Clone(m.sessions), nil }

func (m *memStore) Delete(id string) error {
	m.sessions = slices.DeleteFunc(m.sessions, func(s *session.Session) bool { return s.ID == id })
	return nil
}

type fixture struct {
	eng   *enginetest.Engine
	store *memStore
	svc   *Service
}

func newFixture() *fixture {
	eng := enginetest.New()
	eng.AddImage("img-base", "alpine:latest")
	store := &memStore{}
	seq := 0
	svc := NewService(eng, store, reaper.New(eng),
		WithClock(func() time.Time { return time.Unix(int64(seq), 0) }),
		WithIDGenerator(func() (string, error) { seq++; return fmt.Sprintf("id%d", seq), nil }),
	)
	return &fixture{eng: eng, store: store, svc: svc}
}

// composeStarts makes compose up create a container, a network and a volume.
func (f *fixture) composeStarts(running bool) {
	f.eng.OnComposeUp = func(e *enginetest.Engine, p session.Compose, _ []string) (int, error) {
		sel := session.Label{Key: session.ComposeProjectLabelKey, Value: p.Project}
		e.AddImage("img-"+p.Project, p.Project+"-web")
		e.AddNetwork("net-"+p.Project, sel)
		e.AddVolume("vol-"+p.Project, sel)
		e.AddContainer(engine.Container{ID: "c-" + p.Project, ImageID: "img-" + p.Project, Running: running}, sel)
		return 0, nil
	}
}

func TestRun_TearsDownAndPropagatesExitCode(t *testing.T) {
	f := newFixture()
	f.eng.ExitCode = 7
	f.eng.OnCreate = func([]string) engine.Container {
		f.eng.AddImage("img-pulled", "nginx:latest")
		f.eng.AddVolume("vol-anon")
		return engine.Container{ID: "c1", ImageID: "img-pulled", Volumes: []string{"vol-anon"}}
	}

	res, err := f.svc.Run(context.Background(), []string{"nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 7 {
		t.Errorf("exit code = %d", res.ExitCode)
	}
	if f.eng.HasContainer("c1") || f.eng.HasImage("img-pulled") || f.eng.HasVolume("vol-anon") {
		t.Errorf("resources survived: %s", res.Report)
	}
	if !f.eng.HasImage("img-base") {
		t.Error("baseline image removed")
	}
	if len(f.store.sessions) != 0 {
		t.Error("session not forgotten")
	}
}

func TestRun_KeepsPreexistingImage(t *testing.T) {
	f := newFixture()
	f.eng.OnCreate = func([]string) engine.Container { return engine.Container{ID: "c1", ImageID: "img-base"} }

	if _, err := f.svc.Run(context.Background(), []string{"alpine"}); err != nil {
		t.Fatal(err)
	}
	if !f.eng.HasImage("img-base") {
		t.Error("pre-existing image removed")
	}
}

func TestUp_DetachedThenDown(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.composeStarts(true)

	res, err := f.svc.Up(ctx, UpRequest{Name: "stack", Compose: session.Compose{Files: []string{"compose.yml"}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Destroyed || len(f.store.sessions) != 1 {
		t.Fatal("detached session should stay active")
	}
	if got := res.Session.Owned.Images; !slices.Equal(got, []string{"img-ephemeral-id1"}) {
		t.Errorf("owned images = %v", got)
	}

	if _, err := f.svc.Up(ctx, UpRequest{Name: "stack"}); !errors.Is(err, ErrNameInUse) {
		t.Errorf("duplicate name: err = %v", err)
	}

	down, err := f.svc.Down(ctx, []string{"stack"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := down[0].Report.String(); got != "1 container, 1 network, 1 volume, 1 image" {
		t.Errorf("report = %q", got)
	}
	if len(f.store.sessions) != 0 {
		t.Error("session not forgotten")
	}
}

func TestSharedImage_RemovedByLastSession(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.eng.OnComposeUp = func(e *enginetest.Engine, p session.Compose, _ []string) (int, error) {
		e.AddImage("img-shared", "nginx:alpine")
		sel := session.Label{Key: session.ComposeProjectLabelKey, Value: p.Project}
		e.AddContainer(engine.Container{ID: "c-" + p.Project, ImageID: "img-shared", Running: true}, sel)
		return 0, nil
	}

	for _, name := range []string{"a", "b"} {
		if _, err := f.svc.Up(ctx, UpRequest{Name: name}); err != nil {
			t.Fatal(err)
		}
	}

	down, err := f.svc.Down(ctx, []string{"a"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !f.eng.HasImage("img-shared") || len(down[0].Report.Skipped) != 1 {
		t.Fatal("image still used by session b must be kept")
	}

	if _, err := f.svc.Down(ctx, []string{"b"}, false); err != nil {
		t.Fatal(err)
	}
	if f.eng.HasImage("img-shared") {
		t.Error("last session did not remove the shared image")
	}
}

func TestUp_FailureTearsDown(t *testing.T) {
	f := newFixture()
	f.eng.OnComposeUp = func(e *enginetest.Engine, p session.Compose, _ []string) (int, error) {
		e.AddNetwork("net", session.Label{Key: session.ComposeProjectLabelKey, Value: p.Project})
		return 1, nil
	}

	res, err := f.svc.Up(context.Background(), UpRequest{})
	if err == nil || !res.Destroyed {
		t.Fatalf("expected failure and teardown, got %v", err)
	}
	if f.eng.HasNetwork("net") || len(f.store.sessions) != 0 {
		t.Error("failed session left resources behind")
	}
}

func TestUp_AttachTearsDownAfterLogs(t *testing.T) {
	f := newFixture()
	f.composeStarts(true)

	res, err := f.svc.Up(context.Background(), UpRequest{Attach: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Destroyed || f.eng.HasContainer("c-ephemeral-id1") {
		t.Error("attached session not destroyed")
	}
}

func TestDown_SessionSelection(t *testing.T) {
	ctx := context.Background()
	f := newFixture()

	if _, err := f.svc.Down(ctx, nil, false); !errors.Is(err, ErrNoSessions) {
		t.Errorf("no sessions: err = %v", err)
	}

	f.composeStarts(true)
	for range 2 {
		if _, err := f.svc.Up(ctx, UpRequest{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Down(ctx, nil, false); !errors.Is(err, ErrSessionRequired) {
		t.Errorf("ambiguous: err = %v", err)
	}
	if _, err := f.svc.Down(ctx, []string{"missing"}, false); !errors.Is(err, session.ErrNotFound) {
		t.Errorf("missing: err = %v", err)
	}

	down, err := f.svc.Down(ctx, nil, true)
	if err != nil || len(down) != 2 {
		t.Errorf("down --all = %d sessions, %v", len(down), err)
	}
}

func TestGC_OnlyRemovesIdleSessions(t *testing.T) {
	ctx := context.Background()
	f := newFixture()

	f.composeStarts(true)
	if _, err := f.svc.Up(ctx, UpRequest{Name: "busy"}); err != nil {
		t.Fatal(err)
	}
	f.composeStarts(false)
	if _, err := f.svc.Up(ctx, UpRequest{Name: "idle"}); err != nil {
		t.Fatal(err)
	}

	removed, err := f.svc.GC(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0].Session.Name != "idle" {
		t.Errorf("gc removed %+v", removed)
	}

	statuses, _ := f.svc.List(ctx)
	if len(statuses) != 1 || statuses[0].Running != 1 {
		t.Errorf("statuses = %+v", statuses)
	}
}
