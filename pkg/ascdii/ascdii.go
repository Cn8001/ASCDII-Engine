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

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/fx"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type (
	Engine        = app.Engine
	Options       = app.Options
	Scene         = scene.Scene
	GameObject    = scene.GameObject
	Component     = scene.Component
	ComponentMeta = scene.Meta
	Transform     = scene.Transform
	Ctx           = scene.Ctx
	Registry      = scene.Registry
	Camera        = view.Camera
	Env           = light.Env
	Vec2          = geom.Vec2
	Vec3          = geom.Vec3
	AABB          = geom.AABB
	Affine        = geom.Affine
	Color         = color.Color
	Material      = draw.Material
	Context       = draw.Context
	Sprite        = draw.Sprite
	Clip          = anim.Clip
	Ease          = anim.Ease
	Rig           = anim.Rig
	Bone          = anim.Bone
	Pose          = anim.Pose
	Motion        = anim.Motion
	Particles     = fx.System
	Key           = input.Key
	Pass          = pipeline.Pass
	Frame         = pipeline.Frame
	Stage         = pipeline.Stage
	Prop          = prop.Prop
	Binder        = prop.Binder
	Module        = project.Module
	Library       = asset.Library
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

func Box(size Vec3, c Color) Component         { return components.NewBox(size, c) }
func Cylinder(r, h float64, c Color) Component { return components.NewCylinder(r, h, c) }
func Cone(r, h float64, c Color) Component     { return components.NewCone(r, h, c) }
func Blob(r float64, c Color) Component        { return components.NewBlob(r, c) }
func Ground(w, d float64, c Color) Component   { return components.NewGround(w, d, c) }
func Water(w, d float64, c Color) Component    { return components.NewWater(w, d, c) }
func Light(radius float64, c Color) Component  { return components.NewLight(radius, c) }
func Emitter() Component                       { return components.NewEmitter() }
func Plane(w, d float64, c Color) Component    { return components.NewPlane(w, d, c) }
func Quad(w, h float64, c Color) Component     { return components.NewQuad(w, h, c) }
func Label(text string) Component              { return components.NewLabel(text) }
func Player() Component                        { return components.NewPlayer() }
func Toggle(key, target, p string) Component   { return components.NewToggle(key, target, p) }
func DayNight() Component                      { return components.NewDayNight() }
func Quit(key string) Component                { return &components.Quit{Press: key} }
func BoxCollider() Component                   { return components.NewBoxCollider() }
func CircleCollider() Component                { return components.NewCircleCollider() }
func Rigidbody() Component                     { return components.NewRigidbody() }
func AudioSource() Component                   { return components.NewAudioSource() }

// CameraComponent makes its object the game camera, span world units across
// the view. Camera itself is the view type a scene falls back to.
func CameraComponent(span float64) Component {
	c := components.NewCamera()
	c.Span = span
	return c
}
func Fire() Particles   { p, _ := fx.Preset("fire"); return p }
func Embers() Particles { p, _ := fx.Preset("embers"); return p }

// ParticlePreset is a named starting point: embers, fire, smoke, sparks,
// rain, snow, magic, fountain or explosion.
func ParticlePreset(name string) (Particles, bool) { return fx.Preset(name) }
