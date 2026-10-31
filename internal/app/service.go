// Package app implements the ephemeral session use cases.
package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/reaper"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

// Errors returned by the service.
var (
	ErrNameInUse       = errors.New("session name already in use")
	ErrNoSessions      = errors.New("no active sessions")
	ErrSessionRequired = errors.New("multiple active sessions: specify which one(s) or use --all")
)

// Store persists sessions.
type Store interface {
	Save(s *session.Session) error
	List() ([]*session.Session, error)
	Delete(id string) error
}

// Reaper tears down the resources of a session.
type Reaper interface {
	Reap(ctx context.Context, s *session.Session) (reaper.Report, error)
}

// Service orchestrates session lifecycles.
type Service struct {
	engine engine.Engine
	store  Store
	reaper Reaper
	now    func() time.Time
	newID  func() (string, error)
}

// Option customizes a Service.
type Option func(*Service)

// WithClock overrides the time source.
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// WithIDGenerator overrides how session IDs are generated.
func WithIDGenerator(newID func() (string, error)) Option {
	return func(s *Service) { s.newID = newID }
}

// NewService wires a Service.
func NewService(e engine.Engine, store Store, r Reaper, opts ...Option) *Service {
	s := &Service{engine: e, store: store, reaper: r, now: time.Now, newID: session.NewID}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Teardown is the outcome of destroying a session.
type Teardown struct {
	Session *session.Session
	Report  reaper.Report
}

// RunResult is the outcome of Run.
type RunResult struct {
	Teardown
	ExitCode int
}

// Run creates a container from docker run style arguments, attaches to it and
// destroys the session once it exits.
func (s *Service) Run(ctx context.Context, args []string) (RunResult, error) {
	sess, err := s.begin(ctx, "", func(id string, baseline session.Resources) *session.Session {
		return session.NewRun(id, "", baseline, s.now())
	})
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{Teardown: Teardown{Session: sess}}

	ctr, err := s.create(ctx, sess, args)
	if err != nil {
		result.Report, err = s.abort(ctx, sess, err)
		return result, err
	}

	code, runErr := s.engine.StartAttached(ctx, ctr)
	result.ExitCode = code
	report, err := s.teardown(ctx, sess)
	result.Report = report
	return result, errors.Join(runErr, err)
}

// UpRequest describes a compose session.
type UpRequest struct {
	Name    string
	Compose session.Compose
	// Args are forwarded to docker compose up.
	Args []string
	// Attach follows the logs and destroys the session when interrupted.
	Attach bool
}

// UpResult is the outcome of Up.
type UpResult struct {
	Teardown
	// Destroyed is true when the session no longer exists.
	Destroyed bool
}

// Up starts a Docker Compose project as an ephemeral session.
func (s *Service) Up(ctx context.Context, req UpRequest) (UpResult, error) {
	project, err := absolutize(req.Compose)
	if err != nil {
		return UpResult{}, err
	}
	sess, err := s.begin(ctx, req.Name, func(id string, baseline session.Resources) *session.Session {
		return session.NewCompose(id, req.Name, project, baseline, s.now())
	})
	if err != nil {
		return UpResult{}, err
	}
	result := UpResult{Teardown: Teardown{Session: sess}}

	if err := s.composeUp(ctx, sess, req.Args); err != nil {
		result.Report, err = s.abort(ctx, sess, err)
		result.Destroyed = true
		return result, err
	}
	if !req.Attach {
		return result, nil
	}

	_, logsErr := s.engine.ComposeLogs(ctx, *sess.Compose)
	result.Report, err = s.teardown(ctx, sess)
	result.Destroyed = err == nil
	return result, errors.Join(logsErr, err)
}

// Down destroys the referenced sessions. Without references it destroys the
// only active session; with all it destroys every session.
func (s *Service) Down(ctx context.Context, refs []string, all bool) ([]Teardown, error) {
	sessions, err := s.selectSessions(refs, all)
	if err != nil {
		return nil, err
	}
	return s.destroy(ctx, sessions)
}

// Status describes an active session.
type Status struct {
	Session    *session.Session
	Containers int
	Running    int
}

// List returns the status of every active session.
func (s *Service) List(ctx context.Context) ([]Status, error) {
	sessions, err := s.store.List()
	if err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(sessions))
	for _, sess := range sessions {
		containers, err := s.engine.Containers(ctx, sess.Selector)
		if err != nil {
			return nil, fmt.Errorf("inspect session %s: %w", sess.ID, err)
		}
		statuses = append(statuses, Status{Session: sess, Containers: len(containers), Running: countRunning(containers)})
	}
	return statuses, nil
}

// GC destroys sessions left behind with no running container, e.g. after the
// plugin was killed before it could clean up.
func (s *Service) GC(ctx context.Context) ([]Teardown, error) {
	statuses, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var stale []*session.Session
	for _, st := range statuses {
		if st.Running == 0 {
			stale = append(stale, st.Session)
		}
	}
	return s.destroy(ctx, stale)
}

type sessionFactory func(id string, baseline session.Resources) *session.Session

// begin snapshots the host and persists a new session before any resource is
// created, so a crash never leaves untracked resources behind.
func (s *Service) begin(ctx context.Context, name string, newSession sessionFactory) (*session.Session, error) {
	if err := s.ensureNameAvailable(name); err != nil {
		return nil, err
	}

	images, err := s.engine.ImageIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("snapshot images: %w", err)
	}
	volumes, err := s.engine.VolumeNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("snapshot volumes: %w", err)
	}
	id, err := s.newID()
	if err != nil {
		return nil, err
	}

	sess := newSession(id, session.Resources{Images: images, Volumes: volumes})
	if err := s.store.Save(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Service) create(ctx context.Context, sess *session.Session, args []string) (engine.Container, error) {
	id, err := s.engine.CreateContainer(ctx, sess.Selector, args)
	if err != nil {
		return engine.Container{}, fmt.Errorf("create container: %w", err)
	}
	ctr, err := s.engine.InspectContainer(ctx, id)
	if err != nil {
		return engine.Container{}, fmt.Errorf("inspect container: %w", err)
	}
	sess.Owned.Merge(ctr.Resources())
	return ctr, s.store.Save(sess)
}

func (s *Service) composeUp(ctx context.Context, sess *session.Session, args []string) error {
	code, err := s.engine.ComposeUp(ctx, *sess.Compose, args)
	if err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("docker compose up exited with code %d", code)
	}

	containers, err := s.engine.Containers(ctx, sess.Selector)
	if err != nil {
		return fmt.Errorf("inspect containers: %w", err)
	}
	for _, c := range containers {
		sess.Owned.Merge(c.Resources())
	}
	return s.store.Save(sess)
}

// abort destroys a session that failed to start and reports the cause.
func (s *Service) abort(ctx context.Context, sess *session.Session, cause error) (reaper.Report, error) {
	report, err := s.teardown(ctx, sess)
	return report, errors.Join(cause, err)
}

// teardown reaps the session and forgets it. It ignores cancellation so that
// cleanup still happens after an interrupt.
func (s *Service) teardown(ctx context.Context, sess *session.Session) (reaper.Report, error) {
	report, err := s.reaper.Reap(context.WithoutCancel(ctx), sess)
	if err != nil {
		return report, fmt.Errorf("session %s was not fully removed (retry with 'docker ephemeral down %s'): %w", sess.ID, sess.ID, err)
	}
	return report, s.store.Delete(sess.ID)
}

func (s *Service) destroy(ctx context.Context, sessions []*session.Session) ([]Teardown, error) {
	results := make([]Teardown, 0, len(sessions))
	var errs []error
	for _, sess := range sessions {
		report, err := s.teardown(ctx, sess)
		results = append(results, Teardown{Session: sess, Report: report})
		errs = append(errs, err)
	}
	return results, errors.Join(errs...)
}

func (s *Service) selectSessions(refs []string, all bool) ([]*session.Session, error) {
	sessions, err := s.store.List()
	if err != nil {
		return nil, err
	}

	switch {
	case all:
		return sessions, nil
	case len(refs) > 0:
		selected := make([]*session.Session, 0, len(refs))
		for _, ref := range refs {
			sess, err := session.Resolve(sessions, ref)
			if err != nil {
				return nil, err
			}
			if !slices.Contains(selected, sess) {
				selected = append(selected, sess)
			}
		}
		return selected, nil
	case len(sessions) == 0:
		return nil, ErrNoSessions
	case len(sessions) == 1:
		return sessions, nil
	default:
		return nil, ErrSessionRequired
	}
}

func (s *Service) ensureNameAvailable(name string) error {
	if name == "" {
		return nil
	}
	sessions, err := s.store.List()
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.Name == name || sess.ID == name {
			return fmt.Errorf("%w: %s", ErrNameInUse, name)
		}
	}
	return nil
}

// absolutize resolves compose paths so the session can be managed from any
// working directory.
func absolutize(c session.Compose) (session.Compose, error) {
	var err error
	abs := func(p string) string {
		if err != nil || p == "" || p == "-" {
			return p
		}
		var res string
		res, err = filepath.Abs(p)
		return res
	}

	out := c
	out.ProjectDir = abs(c.ProjectDir)
	out.Files = make([]string, len(c.Files))
	for i, f := range c.Files {
		out.Files[i] = abs(f)
	}
	out.EnvFiles = make([]string, len(c.EnvFiles))
	for i, f := range c.EnvFiles {
		out.EnvFiles[i] = abs(f)
	}
	if err != nil {
		return session.Compose{}, fmt.Errorf("resolve compose paths: %w", err)
	}
	return out, nil
}

func countRunning(containers []engine.Container) int {
	n := 0
	for _, c := range containers {
		if c.Running {
			n++
		}
	}
	return n
}
