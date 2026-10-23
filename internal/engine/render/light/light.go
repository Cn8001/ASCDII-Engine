// Package light describes the environment and evaluates irradiance for the
// deferred lighting pass. Lights never touch geometry code.
package light

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

type Sun struct {
	Dir       geom.Vec3   `json:"dir"` // points toward the sun
	Color     color.Color `json:"color"`
	Intensity float64     `json:"intensity"`
}

type Ambient struct {
	Sky       color.Color `json:"sky"`
	Ground    color.Color `json:"ground"`
	Intensity float64     `json:"intensity"`
}

type Point struct {
	Pos       geom.Vec3   `json:"pos"`
	Radius    float64     `json:"radius"`
	Color     color.Color `json:"color"`
	Intensity float64     `json:"intensity"`
	Flicker   float64     `json:"flicker"`
	Dir       geom.Vec3   `json:"dir,omitempty"`  // a spot light's aim; zero shines all around
	Cone      float64     `json:"cone,omitempty"` // cosine of the spot's half angle
}

type Env struct {
	Hour     float64     `json:"hour"`
	Auto     bool        `json:"auto"` // derive sun, ambient and sky from Hour
	Sun      Sun         `json:"sun"`
	Ambient  Ambient     `json:"ambient"`
	Sky      color.Color `json:"sky"`
	Fog      color.Color `json:"fog"`
	Exposure float64     `json:"exposure"`
}

func DefaultEnv() Env {
	e := Env{Hour: 16.5, Auto: true, Exposure: 1}
	return e.Resolved()
}

func (e *Env) bind() *prop.Binder {
	return prop.Bind().
		Float("hour", &e.Hour, 0, 24).
		Bool("auto", &e.Auto).
		Float("exposure", &e.Exposure, 0, 3).
		Vec3("sun.dir", &e.Sun.Dir).
		Color("sun.color", &e.Sun.Color).
		Float("sun.intensity", &e.Sun.Intensity, 0, 3).
		Color("ambient.sky", &e.Ambient.Sky).
		Color("ambient.ground", &e.Ambient.Ground).
		Float("ambient.intensity", &e.Ambient.Intensity, 0, 3).
		Color("sky", &e.Sky).
		Color("fog", &e.Fog)
}

func (e *Env) Props() []prop.Prop           { return e.bind().Props() }
func (e *Env) Set(name string, v any) error { return e.bind().Set(name, v) }

// Resolved fills sun, ambient and sky from the hour when Auto is set.
func (e Env) Resolved() Env {
	if !e.Auto {
		return e
	}
	e.Sun, e.Ambient, e.Sky, e.Fog = Daylight(e.Hour)
	return e
}

// Daylight blends a sun arc with a dimmer, cooler moon so that night scenes
// stay readable instead of collapsing to black dots.
func Daylight(hour float64) (Sun, Ambient, color.Color, color.Color) {
	h := math.Mod(math.Mod(hour, 24)+24, 24)
	az := (h - 6) / 12 * math.Pi
	day := geom.Smoothstep(-.12, .3, math.Sin(az))
	sunDir := arc(az, 62)
	moonDir := arc(az+math.Pi, 48)
	warm := geom.Smoothstep(.0, .55, math.Sin(az))
	sun := Sun{
		Dir:       moonDir.Lerp(sunDir, day).Norm(),
		Color:     color.Mix(color.RGB(170, 182, 205), color.Mix(color.RGB(255, 168, 96), color.RGB(255, 244, 222), warm), day),
		Intensity: geom.Lerp(.42, 1.05, day),
	}
	amb := Ambient{
		Sky:       color.Mix(color.RGB(78, 88, 112), color.Mix(color.RGB(214, 160, 128), color.RGB(190, 212, 222), warm), day),
		Ground:    color.Mix(color.RGB(34, 36, 40), color.RGB(96, 88, 70), day),
		Intensity: geom.Lerp(.5, .58, day),
	}
	sky := color.Mix(color.RGB(12, 14, 20), color.Mix(color.RGB(150, 96, 74), color.RGB(122, 164, 184), warm), day)
	fog := color.Mix(color.RGB(24, 28, 36), color.Mix(color.RGB(176, 128, 104), color.RGB(170, 190, 196), warm), day)
	return sun, amb, sky, fog
}

func arc(az, maxElevDeg float64) geom.Vec3 {
	el := math.Max(.12, math.Sin(az)) * maxElevDeg * math.Pi / 180
	return geom.V3(math.Cos(az)*math.Cos(el), .55*math.Cos(el), math.Sin(el)).Norm()
}

// Shader is the per frame light state, prepared once and evaluated per dot.
type Shader struct {
	sun    geom.Vec3
	sunR   float64
	sunG   float64
	sunB   float64
	skyR   float64
	skyG   float64
	skyB   float64
	grR    float64
	grG    float64
	grB    float64
	points []shadedPoint
	exp    float64
}

type shadedPoint struct {
	pos     geom.Vec3
	radius  float64
	r, g, b float64
	dir     geom.Vec3
	cone    float64
}

func NewShader(env Env, points []Point, time float64) Shader {
	e := env.Resolved()
	s := Shader{sun: e.Sun.Dir.Norm(), exp: e.Exposure}
	if s.exp == 0 {
		s.exp = 1
	}
	s.sunR, s.sunG, s.sunB = scaled(e.Sun.Color, e.Sun.Intensity)
	s.skyR, s.skyG, s.skyB = scaled(e.Ambient.Sky, e.Ambient.Intensity)
	s.grR, s.grG, s.grB = scaled(e.Ambient.Ground, e.Ambient.Intensity)
	for _, p := range points {
		if p.Radius <= 0 {
			continue
		}
		k := p.Intensity
		if p.Flicker > 0 {
			k *= 1 + p.Flicker*(.55*math.Sin(time*11+p.Pos.X)+.35*math.Sin(time*23.7+p.Pos.Y*1.7))
		}
		r, g, b := scaled(p.Color, k)
		sp := shadedPoint{pos: p.Pos, radius: p.Radius, r: r, g: g, b: b}
		if p.Dir != (geom.Vec3{}) {
			sp.dir, sp.cone = p.Dir.Norm(), p.Cone
		}
		s.points = append(s.points, sp)
	}
	return s
}

func scaled(c color.Color, k float64) (float64, float64, float64) {
	r, g, b := c.Float()
	return r * k, g * k, b * k
}

// Irradiance returns the light multiplier for a surface; shadow is 0..1.
func (s Shader) Irradiance(n, world geom.Vec3, shadow float64) (r, g, b float64) {
	t := n.Z*.5 + .5
	r = s.grR + (s.skyR-s.grR)*t
	g = s.grG + (s.skyG-s.grG)*t
	b = s.grB + (s.skyB-s.grB)*t
	if d := n.Dot(s.sun); d > 0 {
		d *= 1 - shadow
		r, g, b = r+s.sunR*d, g+s.sunG*d, b+s.sunB*d
	}
	for _, p := range s.points {
		delta := p.pos.Sub(world)
		dist := delta.Len()
		if dist >= p.radius {
			continue
		}
		// Smooth falloff keeps most of the light inside the radius and still reaches zero at its edge.
		att := 1 - (dist*dist)/(p.radius*p.radius)
		att *= att
		if p.dir != (geom.Vec3{}) && dist > 1e-6 {
			// A soft rim on the cone reads better in Braille than a hard edge.
			att *= geom.Smoothstep(p.cone, p.cone+(1-p.cone)*.25, -delta.Dot(p.dir)/dist)
		}
		wrap := 1.0
		if dist > 1e-6 {
			wrap = (n.Dot(delta.Scale(1/dist)) + .6) / 1.6
		}
		if wrap > 0 {
			k := att * wrap
			r, g, b = r+p.r*k, g+p.g*k, b+p.b*k
		}
	}
	return r * s.exp, g * s.exp, b * s.exp
}
