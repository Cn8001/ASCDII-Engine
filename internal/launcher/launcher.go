package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/term"
)

type Deps struct {
	Version   string
	Demo      fs.FS
	Templates map[string]fs.FS // nil value means the empty starter project
	Recent    recent.Store
	Play      func(ctx context.Context, fsys fs.FS) error
	Help      string
}

// screen ids drive what Enter does in each menu.
const (
	home = iota
	openList
	openPath
	newName
	newDir
	newTemplate
	help
)

type launcher struct {
	d      Deps
	stack  []*Menu
	kinds  []int
	status string
	draft  struct{ name, dir string }
	start  time.Time
}

// Run shows the menu in the terminal until the user quits.
func Run(ctx context.Context, in *os.File, out io.Writer, d Deps) error {
	fd := int(in.Fd())
	if _, _, err := term.Size(fd); err != nil {
		return fmt.Errorf("the menu needs a terminal: %w", err)
	}
	l := &launcher{d: d, start: time.Now()}
	l.push(home, l.homeMenu(ctx))
	for {
		action, err := l.loop(ctx, in, out, fd)
		if err != nil || action == nil {
			return err
		}
		// Games take over the terminal; the menu resumes when they quit.
		if err := action(ctx); err != nil {
			l.status = err.Error()
		}
	}
}

func (l *launcher) push(kind int, m *Menu) {
	l.stack = append(l.stack, m)
	l.kinds = append(l.kinds, kind)
}

func (l *launcher) pop() bool {
	if len(l.stack) <= 1 {
		return false
	}
	l.stack, l.kinds = l.stack[:len(l.stack)-1], l.kinds[:len(l.kinds)-1]
	return true
}

func (l *launcher) top() (*Menu, int) { return l.stack[len(l.stack)-1], l.kinds[len(l.kinds)-1] }

// loop runs the interactive screen and returns an action to perform outside
// raw mode, or nil to quit.
func (l *launcher) loop(ctx context.Context, in *os.File, out io.Writer, fd int) (func(context.Context) error, error) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, state)
	pr := term.NewPresenter(out)
	if err := pr.Enter(ctx); err != nil {
		return nil, err
	}
	defer pr.Leave(context.WithoutCancel(ctx))
	kctx, cancel := context.WithCancel(ctx)
	defer cancel()
	keys := term.Keys(kctx, in)
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()
	g := cell.New(80, 24)
	for {
		select {
		case <-ctx.Done():
			return nil, nil
		case k, ok := <-keys:
			if !ok || k == input.CtrlC {
				return nil, nil
			}
			action, quit := l.key(ctx, k)
			if quit {
				return nil, nil
			}
			if action != nil {
				return action, nil
			}
		case <-tick.C:
			if w, h, err := term.Size(fd); err == nil {
				g.Resize(w, h)
			}
			m, _ := l.top()
			Render(g, m, l.d.Version, l.status, time.Since(l.start).Seconds())
			if err := pr.Present(ctx, g); err != nil {
				return nil, err
			}
		}
	}
}

func (l *launcher) key(ctx context.Context, k input.Key) (func(context.Context) error, bool) {
	m, kind := l.top()
	switch m.Key(k) {
	case Back:
		l.status = ""
		return nil, !l.pop()
	case Chosen:
		l.status = ""
		return l.choose(ctx, kind, m.Selected())
	}
	return nil, false
}

func (l *launcher) homeMenu(ctx context.Context) *Menu {
	items := []Item{
		{Label: "Play Dragonfall", Hint: "demo", Value: "demo"},
		{Label: "Open project", Value: "open"},
		{Label: "New project", Value: "new"},
		{Label: "Help", Value: "help"},
		{Label: "Quit", Value: "quit"},
	}
	if list, err := l.d.Recent.Load(ctx); err == nil && len(list) > 0 {
		items = append([]Item{{Label: "Continue " + projectName(list[0]), Hint: filepath.Dir(list[0]), Value: "path:" + list[0]}}, items...)
	}
	return &Menu{Title: "MAIN MENU", Items: items}
}

func projectName(manifest string) string {
	return strings.TrimSuffix(filepath.Base(manifest), project.Ext)
}

func (l *launcher) choose(ctx context.Context, kind int, it Item) (func(context.Context) error, bool) {
	switch kind {
	case home:
		switch {
		case it.Value == "quit":
			return nil, true
		case it.Value == "demo":
			return func(ctx context.Context) error { return l.d.Play(ctx, l.d.Demo) }, false
		case it.Value == "open":
			l.push(openList, l.openMenu(ctx))
		case it.Value == "new":
			l.push(newName, &Menu{Title: "NEW PROJECT  name", Prompt: true, Text: "My Game"})
		case it.Value == "help":
			l.push(help, l.helpMenu())
		case strings.HasPrefix(it.Value, "path:"):
			return l.playPath(strings.TrimPrefix(it.Value, "path:")), false
		}
	case openList:
		if it.Value == "type" {
			wd, _ := os.Getwd()
			l.push(openPath, &Menu{Title: "OPEN  folder or .ascdiiproj path", Prompt: true, Text: wd + string(os.PathSeparator)})
			return nil, false
		}
		return l.playPath(it.Value), false
	case openPath:
		return l.playPath(expand(it.Value)), false
	case newName:
		l.draft.name = it.Value
		home, _ := os.UserHomeDir()
		l.push(newDir, &Menu{Title: "NEW PROJECT  location", Prompt: true, Text: filepath.Join(home, "ascdii", project.FileName(it.Value))})
	case newDir:
		l.draft.dir = expand(it.Value)
		items := []Item{{Label: "Dragonfall", Hint: "sample valley", Value: "dragonfall"}, {Label: "Empty", Hint: "starter scene", Value: "empty"}}
		l.push(newTemplate, &Menu{Title: "NEW PROJECT  template", Items: items})
	case newTemplate:
		manifest, err := project.Create(ctx, l.draft.dir, l.draft.name, l.d.Templates[it.Value])
		if err != nil {
			l.status = err.Error()
			return nil, false
		}
		_, _ = l.d.Recent.Add(ctx, manifest)
		l.stack, l.kinds = l.stack[:1], l.kinds[:1]
		l.stack[0] = l.homeMenu(ctx)
		return l.playPath(manifest), false
	case help:
		l.pop()
	}
	return nil, false
}

func (l *launcher) openMenu(ctx context.Context) *Menu {
	var items []Item
	if list, err := l.d.Recent.Load(ctx); err == nil {
		for _, p := range list {
			items = append(items, Item{Label: projectName(p), Hint: filepath.Dir(p), Value: p})
		}
	}
	items = append(items, Item{Label: "Type a path", Value: "type"})
	return &Menu{Title: "OPEN PROJECT", Items: items}
}

func (l *launcher) helpMenu() *Menu {
	var items []Item
	for _, line := range strings.Split(strings.TrimSpace(l.d.Help), "\n") {
		items = append(items, Item{Label: strings.TrimSpace(line)})
	}
	return &Menu{Title: "COMMANDS", Items: items}
}

// playPath accepts a project folder or its manifest.
func (l *launcher) playPath(p string) func(context.Context) error {
	return func(ctx context.Context) error {
		dir := p
		if strings.HasSuffix(p, project.Ext) {
			dir = filepath.Dir(p)
		}
		fsys := os.DirFS(dir)
		file, err := project.Find(fsys)
		if err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		if _, err := l.d.Recent.Add(ctx, filepath.Join(dir, file)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return l.d.Play(ctx, fsys)
	}
}

func expand(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
