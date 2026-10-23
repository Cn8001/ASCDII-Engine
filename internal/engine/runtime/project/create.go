package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Create writes a new project into dir. With a template every file is copied
// and the manifest renamed; without one a small starter scene is generated.
func Create(ctx context.Context, dir, name string, template fs.FS) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create project: %w", err)
	}
	manifest := filepath.Join(dir, FileName(name)+Ext)
	if _, err := os.Stat(manifest); err == nil {
		return "", fmt.Errorf("create project: %w", fs.ErrExist)
	}
	if template != nil {
		if err := copyTemplate(ctx, dir, name, template); err != nil {
			return "", err
		}
		_, err := EnsureMeta(ctx, dir, guid.Default())
		return manifest, err
	}
	for _, d := range []string{"scenes"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return "", fmt.Errorf("create project: %w", err)
		}
	}
	var buf bytes.Buffer
	if err := scene.Encode(&buf, Starter(name)); err != nil {
		return "", err
	}
	if err := writeFile(ctx, filepath.Join(dir, "scenes", "main.scene.json"), buf.Bytes()); err != nil {
		return "", err
	}
	if err := writeRepoFiles(ctx, dir, name); err != nil {
		return "", err
	}
	if _, err := EnsureMeta(ctx, dir, guid.Default()); err != nil {
		return "", err
	}
	return manifest, SaveManifest(ctx, manifest, Manifest{Format: 1, Name: name, Version: "0.0.1", Engine: version.Engine, Start: "scenes/main.scene.json", Cols: 120, Rows: 40, FPS: 30})
}

// writeRepoFiles makes a new project ready to commit: build output stays out
// of version control.
func writeRepoFiles(ctx context.Context, dir, name string) error {
	if err := writeFile(ctx, filepath.Join(dir, ".gitignore"), []byte(BuildDir+"/\n")); err != nil {
		return err
	}
	readme := "# " + name + "\n\nAn ASCDII Engine game. Open the `" + Ext + "` file in the editor, or:\n\n" +
		"```bash\nascdii run .\nascdii release .\n```\n"
	return writeFile(ctx, filepath.Join(dir, "README.md"), []byte(readme))
}

func copyTemplate(ctx context.Context, dir, name string, template fs.FS) error {
	src, err := Find(template)
	if err != nil {
		return err
	}
	m, err := ReadManifest(template, src)
	if err != nil {
		return err
	}
	err = fs.WalkDir(template, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || p == src {
			return err
		}
		b, err := fs.ReadFile(template, p)
		if err != nil {
			return err
		}
		return writeFile(ctx, filepath.Join(dir, filepath.FromSlash(p)), b)
	})
	if err != nil {
		return fmt.Errorf("copy template: %w", err)
	}
	m.Name, m.Format, m.Engine = name, 1, version.Engine
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		if err := writeRepoFiles(ctx, dir, name); err != nil {
			return err
		}
	}
	return SaveManifest(ctx, filepath.Join(dir, FileName(name)+Ext), m)
}

func SaveManifest(ctx context.Context, file string, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return writeFile(ctx, file, append(b, '\n'))
}

func writeFile(ctx context.Context, file string, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	if err := os.WriteFile(file, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}

// FileName is the manifest base name for a game name.
func FileName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '/' || r == '\\' || r == ':' {
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
	if clean == "" {
		return "game"
	}
	return clean
}

// Starter is the scene a new empty project opens with.
func Starter(name string) *scene.Scene {
	s := scene.New(name)
	s.Camera.Span = 40
	return s
}
