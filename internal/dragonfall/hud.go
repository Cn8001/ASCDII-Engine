package dragonfall

import (
	"fmt"
	"unicode/utf8"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/realm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// HUD is screen space text: a top bar, a key strip, a minimap and help.
type HUD struct {
	Title    string
	ShowMap  bool
	ShowHelp bool
	Seed     int
	cache    *realm.Cache
	player   geom.Vec3
	hour     float64
}

func (*HUD) Kind() string { return "df.hud" }
func (h *HUD) bind() *prop.Binder {
	return prop.Bind().String("title", &h.Title).Bool("showMap", &h.ShowMap).Bool("showHelp", &h.ShowHelp).Int("seed", &h.Seed, 0, 99999)
}
func (h *HUD) Props() []prop.Prop        { return h.bind().Props() }
func (h *HUD) Set(n string, v any) error { return h.bind().Set(n, v) }

func (h *HUD) Tick(c *scene.Ctx, _ *scene.GameObject, _ float64) {
	h.hour = c.Scene.Env.Hour
	h.player = c.Scene.View().Target
	c.Scene.Walk(func(e *scene.GameObject, _ bool) bool {
		if e.Component("player") != nil {
			h.player = e.Local.Pos
		}
		return true
	})
}

func hudColors() (bg, fg, accent, dim color.Color) {
	return color.RGB(16, 16, 16), color.RGB(222, 218, 206), color.RGB(232, 176, 75), color.RGB(128, 124, 116)
}

func (h *HUD) DrawCells(f *pipeline.Frame, _ scene.Transform) {
	g := f.Cells
	bg, fg, accent, dim := hudColors()
	g.Fill(0, 0, g.W, 1, cell.Cell{Rune: ' ', BG: bg})
	g.Fill(0, g.H-1, g.W, 1, cell.Cell{Rune: ' ', BG: bg})
	g.Text(2, 0, "◆ "+h.Title, accent, bg)
	phase := "DAY"
	if components.IsNight(h.hour) {
		phase = "NIGHT"
	}
	right := fmt.Sprintf("%s  %d,%d  %s", realm.Zone(h.player.X, h.player.Y), int(h.player.X), int(h.player.Y), phase)
	if w := utf8.RuneCountInString(right); g.W > w+len(h.Title)+8 {
		g.Text(g.W-w-2, 0, right, fg, bg)
	}
	keys := "WASD move  SPACE jump  E dragon  T time  M map  H help  Q quit"
	if g.W < len(keys)+4 {
		keys = "WASD  SPACE  E  T  M  H  Q"
	}
	g.Text(2, g.H-1, keys, dim, bg)
	if h.ShowMap && g.W >= 60 && g.H >= 20 {
		h.drawMap(g)
	}
	if h.ShowHelp {
		drawHelp(g)
	}
}

func (h *HUD) drawMap(g *cell.Grid) {
	bg, _, accent, _ := hudColors()
	r := h.cache.Get(uint32(h.Seed))
	left, top := g.W-29, 2
	g.Fill(left, top, 28, 12, cell.Cell{Rune: ' ', BG: bg})
	g.Text(left+1, top, "MAP", accent, bg)
	for yy := 0; yy < 9; yy++ {
		for xx := 0; xx < 25; xx++ {
			wx, wy := int(h.player.X)+(xx-12)*3, int(h.player.Y)+(yy-4)*3
			ch, col := '.', color.RGB(94, 144, 109)
			switch r.At(wx, wy).Kind {
			case realm.Wood:
				ch = '^'
			case realm.Path:
				ch, col = ':', color.RGB(160, 138, 100)
			case realm.Crag:
				ch, col = '#', color.RGB(140, 140, 146)
			case realm.Grove:
				ch, col = '*', color.RGB(120, 170, 200)
			case realm.River, realm.Bridge:
				ch, col = '~', color.RGB(105, 188, 212)
			}
			if xx == 12 && yy == 4 {
				ch, col = '@', accent
			}
			g.Set(left+1+xx, top+2+yy, cell.Cell{Rune: ch, FG: col, BG: bg})
		}
	}
}

func drawHelp(g *cell.Grid) {
	bg, fg, accent, _ := hudColors()
	lines := []string{"DRAGONFALL", "", "WASD   walk", "SPACE  jump", "E      wake dragon", "T      day / night", "M      map", "H      close", "Q      quit"}
	w, h := 26, len(lines)+2
	left, top := (g.W-w)/2, (g.H-h)/2
	g.Fill(left, top, w, h, cell.Cell{Rune: ' ', BG: bg})
	for i, l := range lines {
		c := fg
		if i == 0 {
			c = accent
		}
		g.Text(left+3, top+1+i, l, c, bg)
	}
}
