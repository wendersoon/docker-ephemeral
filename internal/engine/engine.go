// Package engine defines the port through which the application talks to a
// container engine.
package engine

import (
	"context"
	"errors"

	"github.com/wendersoon/docker-ephemeral/internal/session"
)

// Sentinel errors that adapters must wrap so callers can react to them.
var (
	ErrNotFound = errors.New("resource not found")
	ErrInUse    = errors.New("resource in use")
)

// Container is the subset of container state the application relies on.
type Container struct {
	ID          string
	Name        string
	ImageID     string
	Running     bool
	Interactive bool
	Volumes     []string
}

// Resources returns the images and named volumes referenced by the container.
func (c Container) Resources() session.Resources {
	r := session.Resources{Volumes: c.Volumes}
	if c.ImageID != "" {
		r.Images = []string{c.ImageID}
	}
	return r
}

// Engine is the set of operations required from a container engine.
type Engine interface {
	// ImageIDs returns the IDs of every image present on the host.
	ImageIDs(ctx context.Context) ([]string, error)
	// VolumeNames returns the names of every volume present on the host.
	VolumeNames(ctx context.Context) ([]string, error)

	// Containers returns every container (running or not) matching selector.
	Containers(ctx context.Context, selector session.Label) ([]Container, error)
	// InspectContainer returns the container identified by id.
	InspectContainer(ctx context.Context, id string) (Container, error)
	// CreateContainer creates a container labeled with label from docker run
	// style arguments and returns its ID.
	CreateContainer(ctx context.Context, label session.Label, args []string) (string, error)
	// StartAttached starts the container attached to the terminal and returns
	// its exit code.
	StartAttached(ctx context.Context, c Container) (int, error)
	// RemoveContainers force-removes containers and their anonymous volumes.
	RemoveContainers(ctx context.Context, ids []string) error

	// Networks returns the IDs of networks matching selector.
	Networks(ctx context.Context, selector session.Label) ([]string, error)
	// RemoveNetwork removes a network.
	RemoveNetwork(ctx context.Context, id string) error

	// Volumes returns the names of volumes matching selector.
	Volumes(ctx context.Context, selector session.Label) ([]string, error)
	// RemoveVolume removes a volume.
	RemoveVolume(ctx context.Context, name string) error

	// ImageTags returns the repository tags pointing to an image.
	ImageTags(ctx context.Context, id string) ([]string, error)
	// ImageInUse reports whether any container uses the image.
	ImageInUse(ctx context.Context, id string) (bool, error)
	// RemoveImage removes an image reference (tag or ID) without forcing.
	RemoveImage(ctx context.Context, ref string) error

	// Compose runs a docker compose subcommand attached to the terminal for
	// the given project and returns its exit code.
	Compose(ctx context.Context, project session.Compose, args []string) (int, error)
}
