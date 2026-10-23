// Package pipeline runs ordered, toggleable render passes over one frame and
// times each of them. Stages keep geometry before lighting before post.
package pipeline

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

var ErrPassNotFound = errors.New("pass not found")

type Stage int

const (
	Geometry Stage = iota
	Lighting
	Post
	Resolve
	Overlay
)

func (s Stage) String() string {
	return [...]string{"geometry", "lighting", "post", "resolve", "overlay"}[s]
}

type Frame struct {
	Buf   *raster.Buffer
	Cells *cell.Grid
	View  view.View
	Time  float64
	Env   light.Env
	Ctx   *draw.Context
}

func NewFrame(cols, rows int) *Frame {
	return &Frame{Buf: raster.New(cols, rows), Cells: cell.New(cols, rows)}
}

func (f *Frame) Resize(cols, rows int) bool {
	a := f.Buf.Resize(cols, rows)
	b := f.Cells.Resize(cols, rows)
	return a || b
}

// Begin binds camera, time and environment for one render.
func (f *Frame) Begin(cam view.Camera, t float64, env light.Env) {
	f.View = cam.View(f.Buf.W, f.Buf.H)
	f.Time = t
	f.Env = env.Resolved()
	f.Ctx = &draw.Context{Buf: f.Buf, View: f.View, Time: t}
}

type Pass interface {
	Name() string
	Stage() Stage
	Run(f *Frame)
}

type funcPass struct {
	name  string
	stage Stage
	fn    func(*Frame)
}

func (p funcPass) Name() string { return p.name }
func (p funcPass) Stage() Stage { return p.stage }
func (p funcPass) Run(f *Frame) { p.fn(f) }

// Func wraps a closure as a pass, the quickest way to add a HUD or effect.
func Func(name string, stage Stage, fn func(*Frame)) Pass { return funcPass{name, stage, fn} }

type item struct {
	pass    Pass
	enabled bool
	last    time.Duration
}

type Info struct {
	Name     string `json:"name"`
	Stage    string `json:"stage"`
	Enabled  bool   `json:"enabled"`
	Micros   int64  `json:"micros"`
	Editable bool   `json:"editable"`
}

type Pipeline struct {
	items []*item
	now   func() time.Time
	total time.Duration
}

func New(now func() time.Time, passes ...Pass) *Pipeline {
	p := &Pipeline{now: now}
	for _, pass := range passes {
		p.Add(pass)
	}
	return p
}

// Add places a pass after the last pass of the same stage.
func (p *Pipeline) Add(pass Pass) {
	it := &item{pass: pass, enabled: true}
	i := sort.Search(len(p.items), func(i int) bool { return p.items[i].pass.Stage() > pass.Stage() })
	p.items = append(p.items, nil)
	copy(p.items[i+1:], p.items[i:])
	p.items[i] = it
}

func (p *Pipeline) find(name string) (int, error) {
	for i, it := range p.items {
		if it.pass.Name() == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%w: %s", ErrPassNotFound, name)
}

func (p *Pipeline) Pass(name string) (Pass, error) {
	i, err := p.find(name)
	if err != nil {
		return nil, err
	}
	return p.items[i].pass, nil
}

func (p *Pipeline) SetEnabled(name string, on bool) error {
	i, err := p.find(name)
	if err != nil {
		return err
	}
	p.items[i].enabled = on
	return nil
}

func (p *Pipeline) Enabled(name string) bool {
	i, err := p.find(name)
	return err == nil && p.items[i].enabled
}

// Move reorders a pass to index, clamped to the span of its own stage.
func (p *Pipeline) Move(name string, index int) error {
	i, err := p.find(name)
	if err != nil {
		return err
	}
	st := p.items[i].pass.Stage()
	lo, hi := len(p.items), -1
	for j, it := range p.items {
		if it.pass.Stage() == st {
			lo, hi = min(lo, j), max(hi, j)
		}
	}
	index = max(lo, min(hi, index))
	it := p.items[i]
	p.items = append(p.items[:i], p.items[i+1:]...)
	p.items = append(p.items[:index], append([]*item{it}, p.items[index:]...)...)
	return nil
}

func (p *Pipeline) Infos() []Info {
	out := make([]Info, len(p.items))
	for i, it := range p.items {
		out[i] = Info{Name: it.pass.Name(), Stage: it.pass.Stage().String(), Enabled: it.enabled, Micros: it.last.Microseconds(), Editable: isEditable(it.pass)}
	}
	return out
}

func (p *Pipeline) Total() time.Duration { return p.total }

func (p *Pipeline) Render(f *Frame) {
	start := p.now()
	for _, it := range p.items {
		if !it.enabled {
			it.last = 0
			continue
		}
		t := p.now()
		it.pass.Run(f)
		it.last = p.now().Sub(t)
	}
	p.total = p.now().Sub(start)
}
