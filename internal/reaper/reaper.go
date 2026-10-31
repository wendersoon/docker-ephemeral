// Package reaper removes every Docker resource that belongs to a session.
package reaper

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

// Engine is the subset of engine.Engine the reaper depends on.
type Engine interface {
	Containers(ctx context.Context, selector session.Label) ([]engine.Container, error)
	RemoveContainers(ctx context.Context, ids []string) error
	Networks(ctx context.Context, selector session.Label) ([]string, error)
	RemoveNetwork(ctx context.Context, id string) error
	Volumes(ctx context.Context, selector session.Label) ([]string, error)
	RemoveVolume(ctx context.Context, name string) error
	ImageTags(ctx context.Context, id string) ([]string, error)
	TagImage(ctx context.Context, id, ref string) error
	ImageInUse(ctx context.Context, id string) (bool, error)
	RemoveImage(ctx context.Context, ref string) error
}

// Kind identifies a type of Docker resource.
type Kind string

// Resource kinds handled by the reaper.
const (
	KindContainer Kind = "container"
	KindNetwork   Kind = "network"
	KindVolume    Kind = "volume"
	KindImage     Kind = "image"
)

// Skip records a resource that was intentionally kept.
type Skip struct {
	Kind   Kind
	ID     string
	Reason string
}

// Report summarizes a reap.
type Report struct {
	Removed map[Kind][]string
	Skipped []Skip
	// Restored lists tags given back to the image they pointed to before the
	// session started.
	Restored []string
}

// Count returns how many resources of kind were removed.
func (r Report) Count(kind Kind) int { return len(r.Removed[kind]) }

// String renders a one-line summary such as "2 containers, 1 image".
func (r Report) String() string {
	var parts []string
	for _, k := range []Kind{KindContainer, KindNetwork, KindVolume, KindImage} {
		if n := r.Count(k); n > 0 {
			parts = append(parts, plural(n, string(k)))
		}
	}
	if len(parts) == 0 {
		return "nothing to remove"
	}
	return strings.Join(parts, ", ")
}

// Reaper tears down sessions.
type Reaper struct {
	engine Engine
}

// New returns a Reaper using e.
func New(e Engine) *Reaper {
	return &Reaper{engine: e}
}

// Reap removes, in dependency order, the containers, networks, volumes and
// images of s.
//
// Only images and volumes recorded as owned when the session started are
// candidates, never those of the baseline, and resources still used outside
// the session are skipped. Tags the session took over from baseline images are
// given back. Removal failures do not stop the reap; they are joined into the
// returned error.
func (r *Reaper) Reap(ctx context.Context, s *session.Session) (Report, error) {
	rep := Report{Removed: map[Kind][]string{}}

	containers, err := r.engine.Containers(ctx, s.Selector)
	if err != nil {
		return rep, fmt.Errorf("list containers: %w", err)
	}
	disposable := s.Disposable()

	errs := []error{r.removeContainers(ctx, &rep, containers)}
	errs = append(errs, r.removeNetworks(ctx, &rep, s.Selector))
	errs = append(errs, r.removeVolumes(ctx, &rep, s.Selector, disposable.Volumes))
	errs = append(errs, r.removeImages(ctx, &rep, disposable.Images, s.BaselineTags))
	return rep, errors.Join(errs...)
}

func (r *Reaper) removeContainers(ctx context.Context, rep *Report, containers []engine.Container) error {
	ids := make([]string, 0, len(containers))
	for _, c := range containers {
		ids = append(ids, c.ID)
	}
	if err := r.engine.RemoveContainers(ctx, ids); err != nil {
		return fmt.Errorf("remove containers: %w", err)
	}
	rep.Removed[KindContainer] = ids
	return nil
}

func (r *Reaper) removeNetworks(ctx context.Context, rep *Report, sel session.Label) error {
	ids, err := r.engine.Networks(ctx, sel)
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}
	var errs []error
	for _, id := range ids {
		errs = append(errs, rep.record(KindNetwork, id, r.engine.RemoveNetwork(ctx, id)))
	}
	return errors.Join(errs...)
}

func (r *Reaper) removeVolumes(ctx context.Context, rep *Report, sel session.Label, owned []string) error {
	labeled, err := r.engine.Volumes(ctx, sel)
	if err != nil {
		return fmt.Errorf("list volumes: %w", err)
	}
	names := slices.Concat(labeled, owned)
	slices.Sort(names)

	var errs []error
	for _, name := range slices.Compact(names) {
		errs = append(errs, rep.record(KindVolume, name, r.engine.RemoveVolume(ctx, name)))
	}
	return errors.Join(errs...)
}

func (r *Reaper) removeImages(ctx context.Context, rep *Report, ids []string, baselineTags map[string]string) error {
	var errs []error
	for _, id := range ids {
		errs = append(errs, rep.record(KindImage, id, r.removeImage(ctx, rep, id, baselineTags)))
	}
	return errors.Join(errs...)
}

// removeImage releases every tag of the image and then deletes it, unless a
// container outside the session still depends on it. A tag that belonged to
// another image before the session started is moved back to that image.
func (r *Reaper) removeImage(ctx context.Context, rep *Report, id string, baselineTags map[string]string) error {
	inUse, err := r.engine.ImageInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return engine.ErrInUse
	}

	tags, err := r.engine.ImageTags(ctx, id)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if previous, ok := baselineTags[tag]; ok && previous != id {
			err := r.engine.TagImage(ctx, previous, tag)
			if err == nil {
				rep.Restored = append(rep.Restored, tag)
				continue
			}
			if !errors.Is(err, engine.ErrNotFound) {
				return err
			}
		}
		if err := r.engine.RemoveImage(ctx, tag); err != nil && !errors.Is(err, engine.ErrNotFound) {
			return err
		}
	}
	if err := r.engine.RemoveImage(ctx, id); err != nil && !errors.Is(err, engine.ErrNotFound) {
		return err
	}
	return nil
}

// record classifies the outcome of a removal: missing resources are ignored,
// resources in use are skipped and anything else is returned as an error.
func (r *Report) record(kind Kind, id string, err error) error {
	switch {
	case err == nil:
		r.Removed[kind] = append(r.Removed[kind], id)
		return nil
	case errors.Is(err, engine.ErrNotFound):
		return nil
	case errors.Is(err, engine.ErrInUse):
		r.Skipped = append(r.Skipped, Skip{Kind: kind, ID: id, Reason: "in use outside the session"})
		return nil
	default:
		return fmt.Errorf("remove %s %s: %w", kind, id, err)
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
