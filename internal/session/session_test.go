package session

import (
	"reflect"
	"testing"
	"time"
)

func TestNewCompose_DerivesProjectFromID(t *testing.T) {
	s := NewCompose("abc123", "", Compose{Files: []string{"compose.yml"}}, Resources{}, time.Now())

	if s.Compose.Project != "ephemeral-abc123" {
		t.Errorf("project = %q", s.Compose.Project)
	}
	want := Label{Key: ComposeProjectLabelKey, Value: "ephemeral-abc123"}
	if s.Selector != want {
		t.Errorf("selector = %v, want %v", s.Selector, want)
	}
}

func TestNewRun_SelectsByLabel(t *testing.T) {
	s := NewRun("abc123", "web", Resources{}, time.Now())

	if got := s.Selector.String(); got != LabelKey+"=abc123" {
		t.Errorf("selector = %q", got)
	}
	if s.DisplayName() != "web" {
		t.Errorf("display name = %q", s.DisplayName())
	}
}

func TestDisposable_ExcludesBaseline(t *testing.T) {
	s := NewRun("id", "", Resources{Images: []string{"old"}, Volumes: []string{"keep"}}, time.Now())
	s.Owned.Merge(Resources{Images: []string{"new", "old"}, Volumes: []string{"keep", "tmp"}})
	s.Owned.Merge(Resources{Images: []string{"new"}})

	want := Resources{Images: []string{"new"}, Volumes: []string{"tmp"}}
	if got := s.Disposable(); !reflect.DeepEqual(got, want) {
		t.Errorf("Disposable() = %+v, want %+v", got, want)
	}
}

func TestNewID(t *testing.T) {
	a, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewID()
	if len(a) != 2*idBytes || a == b {
		t.Errorf("unexpected ids %q and %q", a, b)
	}
}
