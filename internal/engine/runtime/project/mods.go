package project

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script/vm"
)

const (
	ModsDir = vm.ModsDir // in a project, for testing; bundle.ModsDir next to a build
	ModMain = "main.lua"
)

// LoadMods reads every mod folder of fsys, the folder that holds them: its
// assets go into lib under mods/<name>/ and its main.lua is returned. A
// folder without main.lua is not a mod; a mod that fails to load is
// reported and skipped.
func LoadMods(ctx context.Context, lib *asset.Library, fsys fs.FS) ([]vm.ModSource, []error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, []error{fmt.Errorf("read mods: %w", err)}
	}
	var (
		out  []vm.ModSource
		errs []error
	)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		sub, err := fs.Sub(fsys, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("mod %s: %w", name, err))
			continue
		}
		main, err := fs.ReadFile(sub, ModMain)
		if err != nil {
			continue
		}
		if err := lib.LoadUnder(ctx, sub, path.Join(vm.ModsDir, name)); err != nil {
			errs = append(errs, fmt.Errorf("mod %s: %w", name, err))
			continue
		}
		out = append(out, vm.ModSource{Name: name, Main: string(main)})
	}
	return out, errs
}
