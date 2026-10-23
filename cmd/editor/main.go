//go:build desktop || dev || bindings

// Command editor is the ASCDII Engine desktop editor. Wails builds always set
// one of these tags, so plain go build and go test skip this package and do
// not need a webview toolchain.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/frontend"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/brand"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/service"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/session"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// packageDir is where .ascdiipack files the user collects are listed from.
func packageDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ascdii", "packages")
}

func run() error {
	if !openDirectly && !launch.FromLauncher() {
		return handOver()
	}
	store, err := recent.Default()
	if err != nil {
		return err
	}
	appCtx := &service.Context{}
	sess := session.New(session.Deps{
		Modules:    []project.Module{dragonfall.Module()},
		Templates:  []session.Template{{Name: "empty"}, {Name: "dragonfall", FS: dragonfall.Template()}},
		Packages:   []pack.Source{dragonfall.Package()},
		PackageDir: packageDir(),
		Players:    players.Default(),
		Binary:     true,
		Emit:       emitter{},
		Now:        time.Now,
	})
	var icon bytes.Buffer
	if err := png.Encode(&icon, brand.Icon(256)); err != nil {
		return fmt.Errorf("window icon: %w", err)
	}
	svc := service.New(service.Deps{Session: sess, Ctx: appCtx, Dialogs: dialogs{}, Recent: store, Reveal: reveal, Edit: edit})
	return wails.Run(&options.App{
		Title:            "ASCDII Engine " + version.Engine,
		Width:            1480,
		Height:           920,
		MinWidth:         980,
		MinHeight:        620,
		BackgroundColour: &options.RGBA{R: 14, G: 14, B: 14, A: 255},
		AssetServer:      &assetserver.Options{Assets: frontend.Dist(), Middleware: frames(sess)},
		OnStartup: func(ctx context.Context) {
			appCtx.Set(ctx)
			// Before the page loads, so it starts with the project already open.
			if len(os.Args) > 1 {
				if _, err := svc.Project.Open(os.Args[1]); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}
			go sess.Run(ctx, 60)
		},
		OnShutdown: func(context.Context) { sess.Close() },
		Bind:       svc.All(),
		// Webview drops stay on, so drags inside the editor keep working.
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true},
		Linux:       &linux.Options{Icon: icon.Bytes(), ProgramName: "ascdii-editor"},
	})
}

// handOver asks ascdii to start the editor, for an editor started directly.
func handOver() error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	cli, err := launch.Find(self, launch.CLI)
	if err != nil {
		return fmt.Errorf("the editor opens from %s: %w", launch.CLI, err)
	}
	return launch.Relaunch(context.Background(), cli)
}

// frames serves the latest frame as raw bytes; the frame event only says a
// new one is ready, which keeps large grids out of JSON. It runs before the
// asset server so the dev server's page fallback never answers it.
func frames(sess *session.Session) assetserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rel, ok := strings.CutPrefix(r.URL.Path, "/asset/"); ok {
				file, err := sess.AssetFile(rel)
				if err != nil {
					http.NotFound(w, r)
					return
				}
				http.ServeFile(w, r, file) // sounds for play mode and asset previews
				return
			}
			if r.URL.Path != "/frame" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(sess.Raw())
		})
	}
}

type emitter struct{}

func (emitter) Emit(ctx context.Context, event string, data any) {
	wruntime.EventsEmit(ctx, event, data)
}

type dialogs struct{}

func (dialogs) OpenFile(ctx context.Context, title string, filters []service.Filter) (string, error) {
	fs := make([]wruntime.FileFilter, len(filters))
	for i, f := range filters {
		fs[i] = wruntime.FileFilter{DisplayName: f.Name, Pattern: f.Pattern}
	}
	path, err := wruntime.OpenFileDialog(ctx, wruntime.OpenDialogOptions{Title: title, Filters: fs})
	if err != nil {
		return "", fmt.Errorf("open dialog: %w", err)
	}
	return path, nil
}

func (dialogs) SaveFile(ctx context.Context, title, name string, filters []service.Filter) (string, error) {
	fs := make([]wruntime.FileFilter, len(filters))
	for i, f := range filters {
		fs[i] = wruntime.FileFilter{DisplayName: f.Name, Pattern: f.Pattern}
	}
	path, err := wruntime.SaveFileDialog(ctx, wruntime.SaveDialogOptions{Title: title, DefaultFilename: name, Filters: fs})
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	return path, nil
}

func (dialogs) PickDir(ctx context.Context, title string) (string, error) {
	dir, err := wruntime.OpenDirectoryDialog(ctx, wruntime.OpenDialogOptions{Title: title, CanCreateDirectories: true})
	if err != nil {
		return "", fmt.Errorf("folder dialog: %w", err)
	}
	return dir, nil
}

// reveal opens a folder in the system file manager; under WSL that is the
// Windows Explorer, since a Linux one is rarely installed.
func reveal(ctx context.Context, path string) error {
	var cmd *exec.Cmd
	switch {
	case runtime.GOOS == "windows":
		cmd = exec.CommandContext(ctx, "explorer", path)
	case runtime.GOOS == "darwin":
		cmd = exec.CommandContext(ctx, "open", path)
	case isWSL():
		win, err := exec.CommandContext(ctx, "wslpath", "-w", path).Output()
		if err != nil {
			return fmt.Errorf("reveal: %w", err)
		}
		cmd = exec.CommandContext(ctx, "explorer.exe", strings.TrimSpace(string(win)))
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", path)
	}
	// Explorer exits non zero even on success, so only a failed start is an error.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reveal: %w", err)
	}
	go cmd.Wait()
	return nil
}

// edit opens a file in the user's code editor. ASCDII_EDITOR may hold a
// command such as "subl {file}:{line}"; otherwise VS Code, then the system
// default for the file type.
func edit(ctx context.Context, path string, line int) error {
	if tpl := os.Getenv("ASCDII_EDITOR"); tpl != "" {
		r := strings.NewReplacer("{file}", path, "{line}", fmt.Sprint(max(1, line)))
		parts := strings.Fields(r.Replace(tpl))
		return start(exec.CommandContext(ctx, parts[0], parts[1:]...))
	}
	if code, err := exec.LookPath("code"); err == nil {
		return start(exec.CommandContext(ctx, code, "-g", fmt.Sprintf("%s:%d", path, max(1, line))))
	}
	return reveal(ctx, path)
}

func start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open editor: %w", err)
	}
	go cmd.Wait()
	return nil
}

func isWSL() bool {
	b, err := os.ReadFile("/proc/version")
	return err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft")
}
