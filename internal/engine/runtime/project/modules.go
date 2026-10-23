package project

import (
	"context"
	"io/fs"
	"regexp"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Unneeded returns the modules a removed package brought that nothing else
// uses: no other installed package names them, and no scene or prefab of
// the project, nor the extra documents such as an unsaved open scene, uses
// one of their component kinds.
func Unneeded(ctx context.Context, fsys fs.FS, mods []Module, removed pack.Manifest, extra ...[]byte) ([]string, error) {
	installed, err := pack.Installed(fsys)
	if err != nil {
		return nil, err
	}
	entries, err := asset.List(ctx, fsys, BuildDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range removed.Modules {
		if slices.ContainsFunc(installed, func(m pack.Manifest) bool { return m.Name != removed.Name && slices.Contains(m.Modules, name) }) {
			continue
		}
		uses := kindPattern(mods, name)
		if uses != nil && used(fsys, entries, uses, extra) {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

// kindPattern matches a stored component of any kind the module registers.
func kindPattern(mods []Module, name string) *regexp.Regexp {
	i := slices.IndexFunc(mods, func(m Module) bool { return m.Name == name })
	if i < 0 {
		return nil
	}
	probe := scene.NewRegistry()
	mods[i].Install(probe, asset.NewLibrary(probe))
	var kinds []string
	for _, k := range probe.Kinds() {
		kinds = append(kinds, regexp.QuoteMeta(k))
	}
	if len(kinds) == 0 {
		return nil
	}
	return regexp.MustCompile(`"kind":\s*"(` + strings.Join(kinds, "|") + `)"`)
}

func used(fsys fs.FS, entries []asset.Entry, uses *regexp.Regexp, extra [][]byte) bool {
	for _, b := range extra {
		if uses.Match(b) {
			return true
		}
	}
	for _, e := range entries {
		if (e.Kind != asset.Scene && e.Kind != asset.Prefab) || strings.HasPrefix(e.Path, pack.Dir+"/") {
			continue
		}
		if b, err := fs.ReadFile(fsys, e.Path); err == nil && uses.Match(b) {
			return true
		}
	}
	return false
}
