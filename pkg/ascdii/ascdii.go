// Package ascdii is the public API of ASCDII Engine. It re-exports the engine
// types and constructors so games never import internal packages.
//
//	e := ascdii.New(ascdii.Options{})
//	e.Scene().Add("crate", ascdii.Box(ascdii.V3(2, 2, 2), ascdii.Hex("#b08050")))
//	err := e.Run(ctx, os.Stdin, os.Stdout)
package ascdii

import (
	"context"
	"io/fs"
	"os"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/fx"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/kit"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

type (
	Engine    = app.Engine
	Options   = app.Options
	Scene     = scene.Scene
	Entity    = scene.Entity
	Node      = scene.Node
	Behaviour = scene.Behaviour
	Transform = scene.Transform
	Ctx       = scene.Ctx
	Registry  = scene.Registry
	Camera    = view.Camera
	Env       = light.Env
	Vec2      = geom.Vec2
	Vec3      = geom.Vec3
	AABB      = geom.AABB
	Affine    = geom.Affine
	Color     = color.Color
	Material  = draw.Material
	Context   = draw.Context
	Sprite    = draw.Sprite
	Clip      = anim.Clip
	Ease      = anim.Ease
	Rig       = anim.Rig
	Bone      = anim.Bone
	Pose      = anim.Pose
	Motion    = anim.Motion
	Particles = fx.Emitter
	Key       = input.Key
	Pass      = pipeline.Pass
	Frame     = pipeline.Frame
	Stage     = pipeline.Stage
	Prop      = prop.Prop
	Binder    = prop.Binder
	Pack      = project.Pack
	Library   = asset.Library
)

const (
	Linear  = anim.Linear
	In      = anim.In
	Out     = anim.Out
	InOut   = anim.InOut
	Step    = anim.Step
	Back    = anim.Back
	Elastic = anim.Elastic

	Geometry = pipeline.Geometry
	Lighting = pipeline.Lighting
	Post     = pipeline.Post
	Overlay  = pipeline.Overlay
)

func New(o Options) *Engine { return app.New(o) }

func Open(ctx context.Context, fsys fs.FS, o Options) (*Engine, error) { return app.Open(ctx, fsys, o) }

// OpenDir loads a project folder containing a .ascdiiproj manifest.
func OpenDir(ctx context.Context, dir string, o Options) (*Engine, error) {
	return app.Open(ctx, os.DirFS(dir), o)
}

func V2(x, y float64) Vec2                            { return geom.V2(x, y) }
func V3(x, y, z float64) Vec3                         { return geom.V3(x, y, z) }
func RGB(r, g, b uint8) Color                         { return color.RGB(r, g, b) }
func Hex(s string) Color                              { return color.Hex(s) }
func M(c Color) Material                              { return draw.M(c) }
func Footprint(base, size Vec3) AABB                  { return geom.Footprint(base, size) }
func Bind() *Binder                                   { return prop.Bind() }
func Func(name string, s Stage, fn func(*Frame)) Pass { return pipeline.Func(name, s, fn) }

func Tween(name string, from, to any, seconds float64, e Ease) Clip {
	return anim.Tween(name, from, to, seconds, e)
}

func Box(size Vec3, c Color) Node            { return kit.NewBox(size, c) }
func Cylinder(r, h float64, c Color) Node    { return kit.NewCylinder(r, h, c) }
func Cone(r, h float64, c Color) Node        { return kit.NewCone(r, h, c) }
func Blob(r float64, c Color) Node           { return kit.NewBlob(r, c) }
func Ground(w, d float64, c Color) Node      { return kit.NewGround(w, d, c) }
func Water(w, d float64, c Color) Node       { return kit.NewWater(w, d, c) }
func Light(radius float64, c Color) Node     { return kit.NewLight(radius, c) }
func Emitter() Node                          { return kit.NewEmitter() }
func Label(text string) Node                 { return kit.NewLabel(text) }
func Player() Behaviour                      { return kit.NewPlayer() }
func Toggle(key, target, p string) Behaviour { return kit.NewToggle(key, target, p) }
func DayNight() Behaviour                    { return kit.NewDayNight() }
func Quit(key string) Behaviour              { return &kit.Quit{Press: key} }
func Fire() Particles                        { return fx.Fire() }
func Embers() Particles                      { return fx.Embers() }
