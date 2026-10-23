package asset

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

type Kind string

const (
	Art    Kind = "art"
	Prefab Kind = "prefab"
	Scene  Kind = "scene"
	Script Kind = "script"
)

const (
	PrefabExt = ".prefab.json"
	SceneExt  = ".scene.json"
	ScriptExt = ".lua"
)

type Entry struct {
	Path string `json:"path"`
	Kind Kind   `json:"kind"`
	Size int64  `json:"size"`
	GUID string `json:"guid"` // empty until a .meta exists
}

func Classify(p string) (Kind, bool) {
	l := strings.ToLower(p)
	switch {
	case strings.HasSuffix(l, PrefabExt):
		return Prefab, true
	case strings.HasSuffix(l, SceneExt):
		return Scene, true
	case strings.HasSuffix(l, ScriptExt):
		return Script, true
	case strings.HasSuffix(l, ".xp"), strings.HasSuffix(l, ".json"):
		return Art, true
	}
	return "", false
}

// List walks a project tree; skip names directories never scanned, such as build output.
func List(ctx context.Context, fsys fs.FS, skip ...string) ([]Entry, error) {
	var out []Entry
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && (strings.HasPrefix(d.Name(), ".") || slices.Contains(skip, p)) {
				return fs.SkipDir
			}
			return nil
		}
		k, ok := Classify(p)
		if !ok {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		e := Entry{Path: p, Kind: k, Size: info.Size()}
		if m, err := ReadMeta(fsys, p); err == nil {
			e.GUID = m.GUID
		}
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Library keeps art and prefab sources in memory so that scene decoding, which
// has no context, never touches the disk.
type Library struct {
	mu      sync.RWMutex
	reg     *scene.Registry
	art     map[string]Surface
	prefabs map[string][]byte
	scripts map[string]source
	byGUID  map[string]string
	byPath  map[string]string
}

type source struct {
	text    string
	version int
}

func NewLibrary(reg *scene.Registry) *Library {
	return &Library{reg: reg, art: map[string]Surface{}, prefabs: map[string][]byte{}, scripts: map[string]source{},
		byGUID: map[string]string{}, byPath: map[string]string{}}
}

// Load reads every art file and prefab from fsys.
func (l *Library) Load(ctx context.Context, fsys fs.FS, skip ...string) error {
	entries, err := List(ctx, fsys, skip...)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.GUID != "" {
			l.SetGUID(e.Path, e.GUID)
		}
		if e.Kind == Scene {
			continue
		}
		b, err := fs.ReadFile(fsys, e.Path)
		if err != nil {
			return fmt.Errorf("load %s: %w", e.Path, err)
		}
		switch e.Kind {
		case Script:
			l.PutScript(e.Path, string(b))
		case Prefab:
			l.PutPrefab(e.Path, b)
		case Art:
			s, err := DecodeArt(e.Path, b)
			if err != nil {
				continue // a stray json that is not art is not fatal
			}
			l.PutArt(e.Path, s)
		}
	}
	return nil
}

func DecodeArt(name string, b []byte) (Surface, error) {
	if strings.HasSuffix(strings.ToLower(name), ".xp") {
		layers, err := LoadXP(bytes.NewReader(b), CP437())
		if err != nil {
			return Surface{}, err
		}
		return Flatten(layers)
	}
	return LoadJSON(bytes.NewReader(b))
}

func (l *Library) SetGUID(p, id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p = path.Clean(p)
	if old, ok := l.byPath[p]; ok {
		delete(l.byGUID, old)
	}
	l.byGUID[id], l.byPath[p] = p, id
}

// Resolve turns a reference into a project path. References are GUIDs; plain
// paths from projects made before .meta files still resolve to themselves.
func (l *Library) Resolve(ref string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if p, ok := l.byGUID[ref]; ok {
		return p
	}
	return path.Clean(ref)
}

// Ref is the reference to store for a path: its GUID when known.
func (l *Library) Ref(p string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if id, ok := l.byPath[path.Clean(p)]; ok {
		return id
	}
	return p
}

// Rename moves every entry of an asset to a new path, keeping its GUID.
func (l *Library) Rename(from, to string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	from, to = path.Clean(from), path.Clean(to)
	if v, ok := l.art[from]; ok {
		l.art[to] = v
		delete(l.art, from)
	}
	if v, ok := l.prefabs[from]; ok {
		l.prefabs[to] = v
		delete(l.prefabs, from)
	}
	if v, ok := l.scripts[from]; ok {
		l.scripts[to] = v
		delete(l.scripts, from)
	}
	if id, ok := l.byPath[from]; ok {
		delete(l.byPath, from)
		l.byPath[to], l.byGUID[id] = id, to
	}
}

func (l *Library) PutArt(name string, s Surface) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.art[path.Clean(name)] = s
}

func (l *Library) PutPrefab(name string, b []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefabs[path.Clean(name)] = b
}

func (l *Library) Art(name string) (Surface, error) {
	p := l.Resolve(name)
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.art[p]
	if !ok {
		return Surface{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s, nil
}

func (l *Library) Prefab(name string) (*scene.Entity, error) {
	p := l.Resolve(name)
	l.mu.RLock()
	b, ok := l.prefabs[p]
	l.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return scene.DecodePrefab(bytes.NewReader(b), l.reg)
}

// PrefabSource returns the raw prefab file, for editing and applying overrides.
func (l *Library) PrefabSource(ref string) (string, []byte, error) {
	p := l.Resolve(ref)
	l.mu.RLock()
	defer l.mu.RUnlock()
	b, ok := l.prefabs[p]
	if !ok {
		return "", nil, fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return p, b, nil
}

func (l *Library) PrefabNames() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.prefabs))
	for k := range l.prefabs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// PutScript bumps the version only when the text changed, which is what
// running scripts poll to hot reload.
func (l *Library) PutScript(name, text string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	k := path.Clean(name)
	old := l.scripts[k]
	if old.text == text && old.version > 0 {
		return
	}
	l.scripts[k] = source{text: text, version: old.version + 1}
}

func (l *Library) Script(name string) (string, int, error) {
	p := l.Resolve(name)
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.scripts[p]
	if !ok {
		return "", 0, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s.text, s.version, nil
}

func (l *Library) ScriptNames() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.scripts))
	for k := range l.scripts {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
