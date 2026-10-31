// Package enginetest provides an in-memory engine.Engine for tests.
package enginetest

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

type labels map[string]string

func (l labels) match(sel session.Label) bool { return l[sel.Key] == sel.Value }

type container struct {
	engine.Container
	labels labels
}

// Engine simulates a container engine host. Its zero value is not usable;
// create one with New.
type Engine struct {
	mu         sync.Mutex
	images     map[string][]string // id -> tags
	volumes    map[string]labels
	networks   map[string]labels
	containers map[string]*container

	// OnCreate is called by CreateContainer to build the new container.
	OnCreate func(args []string) engine.Container
	// OnComposeUp is called by ComposeUp; it may mutate the engine.
	OnComposeUp func(e *Engine, project session.Compose, args []string) (int, error)
	// ExitCode is returned by StartAttached.
	ExitCode int
}

var _ engine.Engine = (*Engine)(nil)

// New returns an empty fake engine.
func New() *Engine {
	return &Engine{
		images:     map[string][]string{},
		volumes:    map[string]labels{},
		networks:   map[string]labels{},
		containers: map[string]*container{},
	}
}

// AddImage registers an image with optional tags.
func (e *Engine) AddImage(id string, tags ...string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.images[id] = tags
}

// AddVolume registers a volume with optional labels.
func (e *Engine) AddVolume(name string, lbl ...session.Label) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.volumes[name] = toLabels(lbl)
}

// AddNetwork registers a network with optional labels.
func (e *Engine) AddNetwork(id string, lbl ...session.Label) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.networks[id] = toLabels(lbl)
}

// AddContainer registers a container with optional labels.
func (e *Engine) AddContainer(c engine.Container, lbl ...session.Label) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.containers[c.ID] = &container{Container: c, labels: toLabels(lbl)}
}

// HasImage reports whether the image exists.
func (e *Engine) HasImage(id string) bool { return has(e, e.images, id) }

// HasVolume reports whether the volume exists.
func (e *Engine) HasVolume(name string) bool { return has(e, e.volumes, name) }

// HasNetwork reports whether the network exists.
func (e *Engine) HasNetwork(id string) bool { return has(e, e.networks, id) }

// HasContainer reports whether the container exists.
func (e *Engine) HasContainer(id string) bool { return has(e, e.containers, id) }

// ImageIDs implements engine.Engine.
func (e *Engine) ImageIDs(context.Context) ([]string, error) { return sortedKeys(e, e.images), nil }

// VolumeNames implements engine.Engine.
func (e *Engine) VolumeNames(context.Context) ([]string, error) { return sortedKeys(e, e.volumes), nil }

// Containers implements engine.Engine.
func (e *Engine) Containers(_ context.Context, sel session.Label) ([]engine.Container, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []engine.Container
	for _, id := range slices.Sorted(maps.Keys(e.containers)) {
		if c := e.containers[id]; c.labels.match(sel) {
			out = append(out, c.Container)
		}
	}
	return out, nil
}

// InspectContainer implements engine.Engine.
func (e *Engine) InspectContainer(_ context.Context, id string) (engine.Container, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	c, ok := e.containers[id]
	if !ok {
		return engine.Container{}, engine.ErrNotFound
	}
	return c.Container, nil
}

// CreateContainer implements engine.Engine.
func (e *Engine) CreateContainer(_ context.Context, label session.Label, args []string) (string, error) {
	c := e.OnCreate(args)
	e.AddContainer(c, label)
	return c.ID, nil
}

// StartAttached implements engine.Engine.
func (e *Engine) StartAttached(context.Context, engine.Container) (int, error) {
	return e.ExitCode, nil
}

// RemoveContainers implements engine.Engine.
func (e *Engine) RemoveContainers(_ context.Context, ids []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, id := range ids {
		delete(e.containers, id)
	}
	return nil
}

// Networks implements engine.Engine.
func (e *Engine) Networks(_ context.Context, sel session.Label) ([]string, error) {
	return matching(e, e.networks, sel), nil
}

// RemoveNetwork implements engine.Engine.
func (e *Engine) RemoveNetwork(_ context.Context, id string) error {
	return remove(e, e.networks, id)
}

// Volumes implements engine.Engine.
func (e *Engine) Volumes(_ context.Context, sel session.Label) ([]string, error) {
	return matching(e, e.volumes, sel), nil
}

// RemoveVolume implements engine.Engine.
func (e *Engine) RemoveVolume(_ context.Context, name string) error {
	e.mu.Lock()
	for _, c := range e.containers {
		if slices.Contains(c.Volumes, name) {
			e.mu.Unlock()
			return engine.ErrInUse
		}
	}
	e.mu.Unlock()
	return remove(e, e.volumes, name)
}

// ImageTags implements engine.Engine.
func (e *Engine) ImageTags(_ context.Context, id string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	tags, ok := e.images[id]
	if !ok {
		return nil, engine.ErrNotFound
	}
	return slices.Clone(tags), nil
}

// ImageInUse implements engine.Engine.
func (e *Engine) ImageInUse(_ context.Context, id string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.imageInUse(id), nil
}

// RemoveImage implements engine.Engine. Removing a tag untags the image and
// deletes it once no tag is left, mirroring docker image rm.
func (e *Engine) RemoveImage(_ context.Context, ref string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	id, isTag := e.resolveImage(ref)
	if id == "" {
		return engine.ErrNotFound
	}
	if e.imageInUse(id) {
		return engine.ErrInUse
	}
	if isTag {
		e.images[id] = slices.DeleteFunc(e.images[id], func(t string) bool { return t == ref })
		if len(e.images[id]) > 0 {
			return nil
		}
	} else if len(e.images[id]) > 1 {
		return engine.ErrInUse
	}
	delete(e.images, id)
	return nil
}

// ComposeUp implements engine.Engine.
func (e *Engine) ComposeUp(_ context.Context, project session.Compose, args []string) (int, error) {
	if e.OnComposeUp == nil {
		return 0, nil
	}
	return e.OnComposeUp(e, project, args)
}

// ComposeLogs implements engine.Engine.
func (e *Engine) ComposeLogs(context.Context, session.Compose) (int, error) {
	return 0, nil
}

func (e *Engine) resolveImage(ref string) (id string, isTag bool) {
	if _, ok := e.images[ref]; ok {
		return ref, false
	}
	for id, tags := range e.images {
		if slices.Contains(tags, ref) {
			return id, true
		}
	}
	return "", false
}

func (e *Engine) imageInUse(id string) bool {
	for _, c := range e.containers {
		if c.ImageID == id {
			return true
		}
	}
	return false
}

func toLabels(lbl []session.Label) labels {
	out := labels{}
	for _, l := range lbl {
		out[l.Key] = l.Value
	}
	return out
}

func has[V any](e *Engine, m map[string]V, key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := m[key]
	return ok
}

func sortedKeys[V any](e *Engine, m map[string]V) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Sorted(maps.Keys(m))
}

func matching(e *Engine, m map[string]labels, sel session.Label) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if m[k].match(sel) {
			out = append(out, k)
		}
	}
	return out
}

func remove[V any](e *Engine, m map[string]V, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := m[key]; !ok {
		return engine.ErrNotFound
	}
	delete(m, key)
	return nil
}
