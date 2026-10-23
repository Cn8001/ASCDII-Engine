package project

import (
	"context"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

// EnsureMeta gives every asset of a project folder a .meta with a UUIDv7.
func EnsureMeta(ctx context.Context, dir string, src guid.Source) ([]string, error) {
	return asset.EnsureMeta(ctx, dir, src, BuildDir)
}

// Migrate rewrites path references in a scene to GUIDs, for projects saved
// before .meta files existed. It returns how many references changed.
func Migrate(s *scene.Scene, lib *asset.Library) int {
	n := 0
	s.Walk(func(e *scene.Entity, _ bool) bool {
		if e.Inherited() {
			return true
		}
		if e.Prefab != "" && !guid.Valid(e.Prefab) {
			if ref := lib.Ref(e.Prefab); ref != e.Prefab {
				e.Prefab = ref
				n++
			}
		}
		for _, p := range e.Props() {
			if p.Type != prop.Asset && p.Type != prop.ScriptRef {
				continue
			}
			v, _ := p.Value.(string)
			if v == "" || guid.Valid(v) {
				continue
			}
			if ref := lib.Ref(v); ref != v && e.Set(p.Name, ref) == nil {
				n++
			}
		}
		return true
	})
	return n
}
