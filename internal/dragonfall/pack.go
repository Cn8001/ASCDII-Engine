// Package dragonfall is the example content pack: the Volume V Dragonfall
// scene rebuilt on the 2.5D engine as node kinds a project can use.
package dragonfall

import (
	"io/fs"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/realm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/vegetation"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
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
	m := func(label string, object bool) scene.Meta {
		return scene.Meta{Category: Name, Label: label, Object: object}
	}
	r.Register("df.terrain", func() scene.Component { return &Terrain{Seed: Seed, Water: -.9, cache: cache} }, m("Terrain", true))
	r.Register("df.forest", func() scene.Component { return &Forest{Seed: Seed, Density: 1, Sway: .35, cache: cache} }, m("Forest", true))
	r.Register("df.grass", func() scene.Component {
		return &Grass{Seed: Seed, Density: .85, Detail: false, cache: cache, field: vegetation.New()}
	}, m("Grass", true))
	r.Register("df.castle", func() scene.Component { return NewCastle() }, m("Castle", true))
	r.Register("df.bridge", func() scene.Component { return NewBridge() }, m("Bridge", true))
	r.Register("df.dragon", func() scene.Component { return NewDragon() }, m("Dragon", true))
	r.Register("df.hero", func() scene.Component { return NewHero() }, m("Hero", true))
	r.Register("df.hud", func() scene.Component { return &HUD{Title: "DRAGONFALL", Seed: Seed, cache: cache} }, m("HUD", true))
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
