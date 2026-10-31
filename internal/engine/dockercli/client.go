package dockercli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

// Client is an engine.Engine backed by the docker CLI.
type Client struct {
	exec Executor
}

var _ engine.Engine = (*Client)(nil)

// New returns a Client running commands through exec.
func New(exec Executor) *Client {
	return &Client{exec: exec}
}

// Images implements engine.Engine.
func (c *Client) Images(ctx context.Context) ([]engine.Image, error) {
	rows, err := c.lines(ctx, "image", "ls", "--all", "--no-trunc", "--format", "{{.ID}} {{.Repository}}:{{.Tag}}")
	if err != nil {
		return nil, err
	}
	return parseImages(rows), nil
}

// parseImages groups "ID REPOSITORY:TAG" rows by image, ignoring untagged rows.
func parseImages(rows []string) []engine.Image {
	tags := map[string][]string{}
	for _, row := range rows {
		id, ref, _ := strings.Cut(row, " ")
		if _, seen := tags[id]; !seen {
			tags[id] = nil
		}
		if ref != "" && !strings.Contains(ref, "<none>") {
			tags[id] = append(tags[id], ref)
		}
	}
	images := make([]engine.Image, 0, len(tags))
	for _, id := range slices.Sorted(maps.Keys(tags)) {
		images = append(images, engine.Image{ID: id, Tags: tags[id]})
	}
	return images
}

// VolumeNames implements engine.Engine.
func (c *Client) VolumeNames(ctx context.Context) ([]string, error) {
	return c.lines(ctx, "volume", "ls", "--quiet")
}

// Containers implements engine.Engine.
func (c *Client) Containers(ctx context.Context, selector session.Label) ([]engine.Container, error) {
	ids, err := c.lines(ctx, "container", "ls", "--all", "--quiet", "--no-trunc", "--filter", "label="+selector.String())
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return c.inspectContainers(ctx, ids...)
}

// InspectContainer implements engine.Engine.
func (c *Client) InspectContainer(ctx context.Context, id string) (engine.Container, error) {
	containers, err := c.inspectContainers(ctx, id)
	if err != nil {
		return engine.Container{}, err
	}
	return containers[0], nil
}

// CreateContainer implements engine.Engine.
func (c *Client) CreateContainer(ctx context.Context, label session.Label, args []string) (string, error) {
	res, err := c.exec.Execute(ctx, Command{
		Args:         slices.Concat([]string{"container", "create", "--label", label.String()}, args),
		ShowProgress: true,
	})
	if err != nil {
		return "", err
	}
	lines := splitLines(res.Stdout)
	if len(lines) == 0 {
		return "", fmt.Errorf("docker container create: no container id returned")
	}
	return lines[len(lines)-1], nil
}

// StartAttached implements engine.Engine.
func (c *Client) StartAttached(ctx context.Context, ctr engine.Container) (int, error) {
	args := []string{"container", "start", "--attach"}
	if ctr.Interactive {
		args = append(args, "--interactive")
	}
	res, err := c.exec.Execute(ctx, Command{Args: append(args, ctr.ID), Attached: true})
	return res.ExitCode, err
}

// RemoveContainers implements engine.Engine.
func (c *Client) RemoveContainers(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return c.run(ctx, slices.Concat([]string{"container", "rm", "--force", "--volumes"}, ids)...)
}

// Networks implements engine.Engine.
func (c *Client) Networks(ctx context.Context, selector session.Label) ([]string, error) {
	return c.lines(ctx, "network", "ls", "--quiet", "--no-trunc", "--filter", "label="+selector.String())
}

// RemoveNetwork implements engine.Engine.
func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return c.run(ctx, "network", "rm", id)
}

// Volumes implements engine.Engine.
func (c *Client) Volumes(ctx context.Context, selector session.Label) ([]string, error) {
	return c.lines(ctx, "volume", "ls", "--quiet", "--filter", "label="+selector.String())
}

// RemoveVolume implements engine.Engine.
func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	return c.run(ctx, "volume", "rm", name)
}

// ImageTags implements engine.Engine.
func (c *Client) ImageTags(ctx context.Context, id string) ([]string, error) {
	res, err := c.exec.Execute(ctx, Command{Args: []string{"image", "inspect", "--format", "{{json .RepoTags}}", id}})
	if err != nil {
		return nil, err
	}
	var tags []string
	if err := json.Unmarshal(bytes.TrimSpace(res.Stdout), &tags); err != nil {
		return nil, fmt.Errorf("decode tags of image %s: %w", id, err)
	}
	return tags, nil
}

// ImageInUse implements engine.Engine.
func (c *Client) ImageInUse(ctx context.Context, id string) (bool, error) {
	ids, err := c.lines(ctx, "container", "ls", "--all", "--quiet", "--filter", "ancestor="+id)
	return len(ids) > 0, err
}

// TagImage implements engine.Engine.
func (c *Client) TagImage(ctx context.Context, id, ref string) error {
	return c.run(ctx, "image", "tag", id, ref)
}

// RemoveImage implements engine.Engine.
func (c *Client) RemoveImage(ctx context.Context, ref string) error {
	return c.run(ctx, "image", "rm", ref)
}

// ComposeUp implements engine.Engine.
func (c *Client) ComposeUp(ctx context.Context, project session.Compose, args []string) (int, error) {
	res, err := c.exec.Execute(ctx, Command{
		Args:           slices.Concat(composeArgs(project), []string{"up", "--detach"}, args),
		Attached:       true,
		StdoutToStderr: true,
	})
	return res.ExitCode, err
}

// ComposeLogs implements engine.Engine.
func (c *Client) ComposeLogs(ctx context.Context, project session.Compose) (int, error) {
	res, err := c.exec.Execute(ctx, Command{
		Args:     slices.Concat(composeArgs(project), []string{"logs", "--follow"}),
		Attached: true,
	})
	return res.ExitCode, err
}

func composeArgs(p session.Compose) []string {
	args := []string{"compose", "--project-name", p.Project}
	if p.ProjectDir != "" {
		args = append(args, "--project-directory", p.ProjectDir)
	}
	for _, f := range p.Files {
		args = append(args, "--file", f)
	}
	for _, f := range p.EnvFiles {
		args = append(args, "--env-file", f)
	}
	for _, profile := range p.Profiles {
		args = append(args, "--profile", profile)
	}
	return args
}

type containerJSON struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	Image string `json:"Image"`
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Config struct {
		OpenStdin bool `json:"OpenStdin"`
	} `json:"Config"`
	Mounts []struct {
		Type string `json:"Type"`
		Name string `json:"Name"`
	} `json:"Mounts"`
}

func (j containerJSON) toContainer() engine.Container {
	ctr := engine.Container{
		ID:          j.ID,
		Name:        strings.TrimPrefix(j.Name, "/"),
		ImageID:     j.Image,
		Running:     j.State.Running,
		Interactive: j.Config.OpenStdin,
	}
	for _, m := range j.Mounts {
		if m.Type == "volume" && m.Name != "" {
			ctr.Volumes = append(ctr.Volumes, m.Name)
		}
	}
	return ctr
}

func (c *Client) inspectContainers(ctx context.Context, ids ...string) ([]engine.Container, error) {
	res, err := c.exec.Execute(ctx, Command{Args: slices.Concat([]string{"container", "inspect"}, ids)})
	if err != nil {
		return nil, err
	}
	var raw []containerJSON
	if err := json.Unmarshal(res.Stdout, &raw); err != nil {
		return nil, fmt.Errorf("decode container inspect: %w", err)
	}
	containers := make([]engine.Container, 0, len(raw))
	for _, r := range raw {
		containers = append(containers, r.toContainer())
	}
	return containers, nil
}

func (c *Client) run(ctx context.Context, args ...string) error {
	_, err := c.exec.Execute(ctx, Command{Args: args})
	return err
}

func (c *Client) lines(ctx context.Context, args ...string) ([]string, error) {
	res, err := c.exec.Execute(ctx, Command{Args: args})
	if err != nil {
		return nil, err
	}
	return splitLines(res.Stdout), nil
}

func splitLines(b []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out = append(out, line)
		}
	}
	return out
}
