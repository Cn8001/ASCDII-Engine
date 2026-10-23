package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnit_ProjectSettingsSaveToTheManifest(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "empty")
	ctx := context.Background()
	if err := os.WriteFile(filepath.Join(dir, "modapi.lua"), []byte("api.event('key')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	p, err := s.ProjectSettings()
	if err != nil {
		t.Fatal(err)
	}
	p.Name, p.FPS, p.Mods, p.ModAPI = "Quest", 60, true, "modapi.lua"

	// Act
	err = s.SetProjectSettings(ctx, p)
	got, _ := s.ProjectSettings()
	raw, _ := os.ReadFile(filepath.Join(dir, s.proj.File))

	// Assert
	if err != nil || got != p || !strings.Contains(string(raw), `"api": "modapi.lua"`) {
		t.Fatalf("err %v got %+v manifest %s", err, got, raw)
	}
}

func TestUnit_ProjectSettingsRejectModsWithoutAPI(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "empty")
	p, _ := s.ProjectSettings()
	p.Mods = true

	// Act
	err := s.SetProjectSettings(context.Background(), p)

	// Assert
	if !errors.Is(err, ErrBadSettings) {
		t.Fatalf("err %v", err)
	}
}

func TestUnit_PlayRunsTheProjectMods(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "empty")
	ctx := context.Background()
	files := map[string]string{
		"modapi.lua":       `api.func("double", function(n) return n * 2 end)`,
		"mods/hi/main.lua": `function on_load() log("double", game.double(21)) end`,
	}
	for name, src := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	p, _ := s.ProjectSettings()
	p.Mods, p.ModAPI = true, "modapi.lua"
	if err := s.SetProjectSettings(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Act
	err := s.Play()

	// Assert
	found := false
	for _, l := range s.Logs() {
		found = found || (l.Source == "mod:hi" && l.Text == "double 42")
	}
	if err != nil || !found {
		t.Fatalf("err %v logs %+v", err, s.Logs())
	}
}
