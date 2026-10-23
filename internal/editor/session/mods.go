package session

import (
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script/vm"
)

// startMods runs the project's own mods folder in play mode, as a built game
// runs the one next to it, so developers test their mod API with real mods.
func (s *Session) startMods() {
	if s.proj == nil || !s.proj.Manifest.Mods.Enabled {
		return
	}
	dir := filepath.Join(s.dir, project.ModsDir)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return
	}
	sources, errs := project.LoadMods(s.appCtx, s.lib, os.DirFS(dir))
	errs = append(errs, vm.StartMods(&s.sctx, s.lib, s.reg, s.proj.Manifest.Mods.API, sources)...)
	for _, err := range errs {
		s.sctx.Logf("error", "mods", "%v", err)
	}
}
