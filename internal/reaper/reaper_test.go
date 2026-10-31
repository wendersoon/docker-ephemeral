package reaper

import (
	"context"
	"testing"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/engine"
	"github.com/wendersoon/docker-ephemeral/internal/engine/enginetest"
	"github.com/wendersoon/docker-ephemeral/internal/session"
)

func TestReap_RemovesOnlySessionResources(t *testing.T) {
	ctx := context.Background()
	eng := enginetest.New()

	// Pre-existing state that must survive.
	eng.AddImage("img-base", "alpine:latest")
	eng.AddVolume("vol-base")
	eng.AddNetwork("net-other")
	baseline := session.Resources{Images: []string{"img-base"}, Volumes: []string{"vol-base"}}

	s := session.NewCompose("s1", "", session.Compose{}, baseline, time.Now())

	// Resources created by the session.
	eng.AddImage("img-new", "app:dev", "app:latest")
	eng.AddVolume("vol-labeled", s.Selector)
	eng.AddVolume("vol-anon")
	eng.AddNetwork("net-session", s.Selector)
	eng.AddContainer(engine.Container{ID: "c1", ImageID: "img-new", Volumes: []string{"vol-anon", "vol-base"}}, s.Selector)
	eng.AddContainer(engine.Container{ID: "c2", ImageID: "img-base"}, s.Selector)

	rep, err := New(eng).Reap(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	for _, gone := range []bool{
		eng.HasContainer("c1"), eng.HasContainer("c2"), eng.HasImage("img-new"),
		eng.HasVolume("vol-labeled"), eng.HasVolume("vol-anon"), eng.HasNetwork("net-session"),
	} {
		if gone {
			t.Errorf("session resource survived: %s", rep)
		}
	}
	if !eng.HasImage("img-base") || !eng.HasVolume("vol-base") || !eng.HasNetwork("net-other") {
		t.Error("baseline resource was removed")
	}
	if got := rep.String(); got != "2 containers, 1 network, 2 volumes, 1 image" {
		t.Errorf("report = %q", got)
	}
}

func TestReap_SkipsImagesUsedElsewhere(t *testing.T) {
	eng := enginetest.New()
	s := session.NewRun("s1", "", session.Resources{}, time.Now())

	eng.AddImage("img-shared", "shared:1")
	eng.AddContainer(engine.Container{ID: "mine", ImageID: "img-shared"}, s.Selector)
	eng.AddContainer(engine.Container{ID: "theirs", ImageID: "img-shared"})

	rep, err := New(eng).Reap(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if !eng.HasImage("img-shared") {
		t.Fatal("shared image was removed")
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0].Kind != KindImage {
		t.Errorf("skipped = %+v", rep.Skipped)
	}
}

func TestReap_UsesOwnedResourcesOfRemovedContainers(t *testing.T) {
	eng := enginetest.New()
	s := session.NewRun("s1", "", session.Resources{}, time.Now())
	s.Owned = session.Resources{Images: []string{"img"}, Volumes: []string{"vol"}}

	// The container is already gone (e.g. started with --rm).
	eng.AddImage("img")
	eng.AddVolume("vol")

	if _, err := New(eng).Reap(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if eng.HasImage("img") || eng.HasVolume("vol") {
		t.Error("owned resources survived")
	}
}

func TestReap_EmptySession(t *testing.T) {
	rep, err := New(enginetest.New()).Reap(context.Background(), session.NewRun("s", "", session.Resources{}, time.Now()))
	if err != nil || rep.String() != "nothing to remove" {
		t.Errorf("got %q, %v", rep, err)
	}
}
