package session

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFileStore_RoundTrip(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	now := time.Now().UTC().Truncate(time.Second)

	newer := NewRun("bbb", "", Resources{}, now)
	older := NewCompose("aaa", "db", Compose{Files: []string{"/x/compose.yml"}}, Resources{Images: []string{"i"}}, now.Add(-time.Minute))
	for _, s := range []*Session{newer, older} {
		if err := store.Save(s); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []*Session{older, newer}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}

	if err := store.Delete("aaa"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("aaa"); err != nil {
		t.Fatalf("deleting twice should succeed: %v", err)
	}
	if got, _ := store.List(); len(got) != 1 {
		t.Errorf("expected one session left, got %d", len(got))
	}
}

func TestFileStore_ListMissingDir(t *testing.T) {
	got, err := NewFileStore(filepath.Join(t.TempDir(), "missing")).List()
	if err != nil || got != nil {
		t.Errorf("List() = %v, %v; want nil, nil", got, err)
	}
}

func TestResolve(t *testing.T) {
	sessions := []*Session{
		{ID: "a1b2c3", Name: "api"},
		{ID: "a1ffff"},
		{ID: "c0ffee", Name: "db"},
	}

	tests := []struct {
		ref     string
		wantID  string
		wantErr error
	}{
		{ref: "api", wantID: "a1b2c3"},
		{ref: "c0ffee", wantID: "c0ffee"},
		{ref: "a1f", wantID: "a1ffff"},
		{ref: "a1", wantErr: ErrAmbiguous},
		{ref: "zz", wantErr: ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got, err := Resolve(sessions, tt.ref)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got.ID != tt.wantID {
				t.Errorf("got %s, want %s", got.ID, tt.wantID)
			}
		})
	}
}
