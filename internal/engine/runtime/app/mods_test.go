package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

// modProject is a new project whose manifest sets mods as given.
func modProject(t *testing.T, mods project.ModSettings) string {
	t.Helper()
	dir := t.TempDir()
	file, err := project.Create(context.Background(), dir, "demo", nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	b, _ := os.ReadFile(file)
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	m["mods"] = mods
	b, _ = json.Marshal(m)
	if err := os.WriteFile(file, b, 0o644); err != nil {
		t.Fatal(err)
	}
	api := `api.func("greet", function(who) return "hello " .. who end)`
	if err := os.WriteFile(filepath.Join(dir, "modapi.lua"), []byte(api), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

var greeter = fstest.MapFS{
	"greeter/main.lua": {Data: []byte(`function on_load() log(game.greet("mods")) end`)},
	"notes/readme.txt": {Data: []byte("not a mod")},
}

func TestUnit_EnabledModsStartOnFirstUpdate(t *testing.T) {
	// Arrange
	dir := modProject(t, project.ModSettings{Enabled: true, API: "modapi.lua"})
	var lines []string
	log := func(level, source, msg string) { lines = append(lines, level+" "+source+" "+msg) }
	e, err := Open(context.Background(), os.DirFS(dir), Options{Mods: greeter, Log: log})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	_, errA := e.Update(.1, nil)
	_, errB := e.Update(.1, nil)

	// Assert
	if errA != nil || errB != nil || !slices.Equal(lines, []string{"info mod:greeter hello mods"}) {
		t.Fatalf("errs %v %v lines %v", errA, errB, lines)
	}
}

func TestUnit_DisabledModsDoNotLoad(t *testing.T) {
	// Arrange
	dir := modProject(t, project.ModSettings{API: "modapi.lua"})
	var lines []string
	log := func(level, source, msg string) { lines = append(lines, msg) }
	e, err := Open(context.Background(), os.DirFS(dir), Options{Mods: greeter, Log: log})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	_, err = e.Update(.1, nil)

	// Assert
	if err != nil || len(lines) != 0 {
		t.Fatalf("err %v lines %v", err, lines)
	}
}
