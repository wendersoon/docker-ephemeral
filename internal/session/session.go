// Package session models an ephemeral session: the set of Docker resources
// created by one workload and destroyed together when it ends.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"
)

const (
	// LabelKey marks containers started by a run session.
	LabelKey = "io.github.wendersoon.ephemeral.session"
	// ComposeProjectLabelKey is set by Docker Compose on every resource it creates.
	ComposeProjectLabelKey = "com.docker.compose.project"

	composeProjectPrefix = "ephemeral-"
	idBytes              = 6
)

// Mode is the kind of workload a session wraps.
type Mode string

// Supported session modes.
const (
	ModeRun     Mode = "run"
	ModeCompose Mode = "compose"
)

// Label is a Docker label used to select resources.
type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// String formats the label as accepted by docker --label and --filter.
func (l Label) String() string { return l.Key + "=" + l.Value }

// Compose holds the Docker Compose project backing a compose session.
type Compose struct {
	Project    string   `json:"project"`
	ProjectDir string   `json:"projectDir,omitempty"`
	Files      []string `json:"files,omitempty"`
	EnvFiles   []string `json:"envFiles,omitempty"`
	Profiles   []string `json:"profiles,omitempty"`
}

// Resources is a set of images and volumes.
type Resources struct {
	Images  []string `json:"images,omitempty"`
	Volumes []string `json:"volumes,omitempty"`
}

// Merge adds other to r, keeping entries unique and sorted.
func (r *Resources) Merge(other Resources) {
	r.Images = union(r.Images, other.Images)
	r.Volumes = union(r.Volumes, other.Volumes)
}

// Without returns the resources of r that are not present in other.
func (r Resources) Without(other Resources) Resources {
	return Resources{
		Images:  difference(r.Images, other.Images),
		Volumes: difference(r.Volumes, other.Volumes),
	}
}

// Session is the persisted state of an ephemeral workload.
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Mode      Mode      `json:"mode"`
	CreatedAt time.Time `json:"createdAt"`
	// Selector matches every container, network and volume of the session.
	Selector Label    `json:"selector"`
	Compose  *Compose `json:"compose,omitempty"`
	// Baseline holds the resources that existed before the session started;
	// they are never removed.
	Baseline Resources `json:"baseline"`
	// BaselineTags maps every tag of the baseline to its image, so a tag moved
	// by the session (e.g. by pulling a newer image) can be given back.
	BaselineTags map[string]string `json:"baselineTags,omitempty"`
	// Owned holds the resources used by the session's containers when it
	// started. Resources first used later are never considered owned.
	Owned Resources `json:"owned"`
}

// NewRun creates a session for a single container.
func NewRun(id, name string, baseline Resources, now time.Time) *Session {
	return &Session{
		ID:        id,
		Name:      name,
		Mode:      ModeRun,
		CreatedAt: now,
		Selector:  Label{Key: LabelKey, Value: id},
		Baseline:  baseline,
	}
}

// NewCompose creates a session for a Docker Compose project. The project name
// is derived from the session ID so that every resource can be traced back.
func NewCompose(id, name string, compose Compose, baseline Resources, now time.Time) *Session {
	compose.Project = composeProjectPrefix + id
	return &Session{
		ID:        id,
		Name:      name,
		Mode:      ModeCompose,
		CreatedAt: now,
		Selector:  Label{Key: ComposeProjectLabelKey, Value: compose.Project},
		Compose:   &compose,
		Baseline:  baseline,
	}
}

// DisplayName returns the name of the session, falling back to its ID.
func (s *Session) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.ID
}

// Disposable returns the owned resources that did not exist before the session.
func (s *Session) Disposable() Resources {
	return s.Owned.Without(s.Baseline)
}

// NewID returns a random, URL-safe session identifier.
func NewID() (string, error) {
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func union(a, b []string) []string {
	out := slices.Concat(a, b)
	slices.Sort(out)
	return slices.Compact(out)
}

func difference(a, b []string) []string {
	var out []string
	for _, v := range a {
		if !slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}
