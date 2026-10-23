package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/session"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type fakeDialogs struct{ file, dir string }

func (f fakeDialogs) OpenFile(context.Context, string, []Filter) (string, error) { return f.file, nil }
func (f fakeDialogs) PickDir(context.Context, string) (string, error)            { return f.dir, nil }
func (f fakeDialogs) SaveFile(context.Context, string, string, []Filter) (string, error) {
	return f.file, nil
}

func services(t *testing.T, d fakeDialogs) *Services {
	t.Helper()
	s := session.New(session.Deps{
		Modules:   []project.Module{dragonfall.Module()},
		Templates: []session.Template{{Name: "empty"}, {Name: "dragonfall", FS: dragonfall.Template()}},
		Players:   players.Source{Dirs: []fs.FS{fstest.MapFS{}}},
	})
	return New(Deps{Session: s, Ctx: &Context{}, Dialogs: d, Recent: recent.Store{Path: filepath.Join(t.TempDir(), "recent.json")},
		Reveal: func(context.Context, string) error { return nil }, Edit: func(context.Context, string, int) error { return nil }})
}

func TestUnit_CodeMapsSentinels(t *testing.T) {
	// Arrange
	err := fmt.Errorf("deep: %w", scene.ErrPrefabLocked)

	// Act
	got, other := Code(err), Code(errors.New("boom"))

	// Assert
	if got != "prefab_locked" || other != "internal" {
		t.Fatalf("got %s %s", got, other)
	}
}

func TestUnit_CreateOpensAndRemembers(t *testing.T) {
	// Arrange
	sv := services(t, fakeDialogs{})
	parent := t.TempDir()

	// Act
	info, err := sv.Project.Create(parent, "Star Port", "dragonfall")
	list, errRecent := sv.Project.Recent()

	// Assert
	if err != nil || errRecent != nil || !info.Open || info.Name != "Star Port" || info.Version != "0.1.0" {
		t.Fatalf("err %v %v info %+v", err, errRecent, info)
	}
	if len(list) != 1 || !strings.HasSuffix(list[0], "Star-Port.ascdiiproj") {
		t.Fatalf("recent %v", list)
	}
}

func TestUnit_ErrorsCarryCodePrefix(t *testing.T) {
	// Arrange
	sv := services(t, fakeDialogs{})

	// Act
	err := sv.Scene.SetProp(999, "x", 1)
	_, errDialog := sv.Project.OpenDialog()

	// Assert
	if err == nil || !strings.HasPrefix(err.Error(), "not_found: ") {
		t.Fatalf("got %v", err)
	}
	if errDialog == nil || !strings.HasPrefix(errDialog.Error(), "cancelled: ") {
		t.Fatalf("got %v", errDialog)
	}
}

func TestUnit_ServicesRoundTrip(t *testing.T) {
	// Arrange
	sv := services(t, fakeDialogs{})
	if _, err := sv.Project.Create(t.TempDir(), "g", "empty"); err != nil {
		t.Fatal(err)
	}

	// Act
	id, err := sv.Scene.Add(0, "cone")
	errSet := sv.Scene.SetProp(id, "radius", 3)
	props, errProps := sv.Scene.Props(id)
	passes := sv.Pipeline.Passes()

	// Assert
	if err != nil || errSet != nil || errProps != nil || len(passes) == 0 {
		t.Fatalf("errs %v %v %v", err, errSet, errProps)
	}
	found := false
	for _, p := range props {
		if p.Name == "cone.radius" && p.Value.(float64) == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("props %+v", props)
	}
}
