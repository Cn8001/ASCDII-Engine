// Package dragonfall is the example content pack: the Volume V Dragonfall
// scene rebuilt on the 2.5D engine as node kinds a project can use.
package dragonfall

import (
	"io/fs"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/realm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/vegetation"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

const (
	Name = "dragonfall"
	Seed = 2026
)

func Pack() project.Pack { return project.Pack{Name: Name, Install: Install} }

// Install registers the pack's kinds; they share one realm cache so terrain,
// forest, grass and HUD agree on the same valley.
func Install(r *scene.Registry, _ *asset.Library) {
	cache := realm.NewCache()
	r.Node("df.terrain", func() scene.Node { return &Terrain{Seed: Seed, Water: -.9, cache: cache} })
	r.Node("df.forest", func() scene.Node { return &Forest{Seed: Seed, Density: 1, Sway: .35, cache: cache} })
	r.Node("df.grass", func() scene.Node {
		return &Grass{Seed: Seed, Density: .85, Detail: false, cache: cache, field: vegetation.New()}
	})
	r.Node("df.castle", func() scene.Node { return NewCastle() })
	r.Node("df.bridge", func() scene.Node { return NewBridge() })
	r.Node("df.dragon", func() scene.Node { return NewDragon() })
	r.Node("df.hero", func() scene.Node { return NewHero() })
	r.Node("df.hud", func() scene.Node { return &HUD{Title: "DRAGONFALL", Seed: Seed, cache: cache} })
}

// Template is the Dragonfall project used by "ascdii new -template dragonfall"
// and the editor's new project dialog.
func Template() fs.FS {
	sub, err := fs.Sub(templateFiles, "template")
	if err != nil {
		panic(err) // the embed pattern guarantees the directory exists
	}
	return sub
}
