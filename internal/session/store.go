package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Errors returned when resolving sessions.
var (
	ErrNotFound  = errors.New("session not found")
	ErrAmbiguous = errors.New("session reference is ambiguous")
)

const fileExt = ".json"

// FileStore persists sessions as JSON files in a directory.
type FileStore struct {
	dir string
}

// NewFileStore returns a store rooted at dir.
func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

// DefaultDir returns the directory where sessions are stored, honoring DOCKER_CONFIG.
func DefaultDir() (string, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		configDir = filepath.Join(home, ".docker")
	}
	return filepath.Join(configDir, "ephemeral", "sessions"), nil
}

// Save writes the session atomically.
func (f *FileStore) Save(s *Session) error {
	if err := os.MkdirAll(f.dir, 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session %s: %w", s.ID, err)
	}

	tmp, err := os.CreateTemp(f.dir, s.ID+".*.tmp")
	if err != nil {
		return fmt.Errorf("save session %s: %w", s.ID, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("save session %s: %w", s.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save session %s: %w", s.ID, err)
	}
	if err := os.Rename(tmp.Name(), f.path(s.ID)); err != nil {
		return fmt.Errorf("save session %s: %w", s.ID, err)
	}
	return nil
}

// List returns every stored session, oldest first.
func (f *FileStore) List() ([]*Session, error) {
	entries, err := os.ReadDir(f.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var sessions []*Session
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != fileExt {
			continue
		}
		s, err := f.load(filepath.Join(f.dir, e.Name()))
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	slices.SortFunc(sessions, func(a, b *Session) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return sessions, nil
}

// Delete removes the session. Deleting a missing session is not an error.
func (f *FileStore) Delete(id string) error {
	if err := os.Remove(f.path(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete session %s: %w", id, err)
	}
	return nil
}

func (f *FileStore) path(id string) string {
	return filepath.Join(f.dir, id+fileExt)
}

func (f *FileStore) load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode session %s: %w", filepath.Base(path), err)
	}
	return &s, nil
}

// Resolve finds the session referenced by ref, which may be a name, a full ID
// or an unambiguous ID prefix.
func Resolve(sessions []*Session, ref string) (*Session, error) {
	var matches []*Session
	for _, s := range sessions {
		if s.ID == ref || s.Name == ref {
			return s, nil
		}
		if strings.HasPrefix(s.ID, ref) {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrNotFound, ref)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrAmbiguous, ref)
	}
}
