// Package realm generates the Dragonfall valley. It is deterministic per seed
// so every node that needs it can rebuild the same tiles.
package realm

import (
	"math"
	"sync"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/noise"
)

type Kind uint8

const (
	Meadow Kind = iota
	Wood
	Path
	River
	Bridge
	Crag
	Grove
	Ruin
)

type Tile struct {
	Kind      Kind
	Elevation int
	Variant   uint32
}

type Realm struct {
	Width, Height int
	Seed          uint32
	cells         []Tile
}

// Outside the map is cliff, which also closes the world for collision.
func (r *Realm) At(x, y int) Tile {
	if x < 0 || y < 0 || x >= r.Width || y >= r.Height {
		return Tile{Kind: Crag, Elevation: 3}
	}
	return r.cells[y*r.Width+x]
}

// RiverAt is the winding Mirrorstream band crossing the valley west to east.
func RiverAt(x, y float64) bool {
	center := 88 + 3*math.Sin(x*.095) + 1.9*math.Sin(x*.033)
	return math.Abs(y-center) < 2.45
}

func BridgeAt(x, y float64) bool { return RiverAt(x, y) && x > 117 && x < 127 }

func New(w, h int, seed uint32) *Realm {
	r := &Realm{Width: w, Height: h, Seed: seed, cells: make([]Tile, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r.cells[y*w+x] = generate(x, y, seed)
		}
	}
	return r
}

func generate(x, y int, seed uint32) Tile {
	fx, fy := float64(x), float64(y)
	grove := noise.Value(fx/32, fy/26, seed)
	rough := noise.Value(fx/13, fy/11, seed+2)
	riverX := 157 + 7*math.Sin(fy/18) + 2*math.Sin(fy/6)
	roadY := 74 + 3*math.Sin(fx/16)
	t := Tile{Kind: Meadow, Variant: noise.Hash(x, y, seed)}
	if grove > .57 || (x < 78 && grove > .43) {
		t.Kind = Wood
	}
	if x > 172 && y < 72 && rough > .5 {
		t.Kind, t.Elevation = Crag, 1+int(rough*3)
	}
	if x > 85 && x < 133 && y > 92 && y < 123 {
		t.Kind = Grove
	}
	if math.Abs(fy-roadY) < 1.7 {
		t.Kind, t.Elevation = Path, 0
	}
	if math.Abs(fx-riverX) < 4 {
		t.Kind, t.Elevation = River, 0
	}
	if y >= 68 && y <= 71 && math.Abs(fx-riverX) < 6.8 {
		t.Kind = Bridge
	}
	if x >= 201 && x < 224 && y >= 33 && y < 53 {
		t.Kind, t.Elevation = Ruin, 1
	}
	if x >= 108 && x < 142 && y >= 54 && y < 92 && t.Kind != River && t.Kind != Bridge {
		dx, dy := (fx-124)/17, (fy-74)/16
		if dx*dx+dy*dy < 1 {
			if t.Kind == Wood || t.Kind == Crag {
				t.Kind, t.Elevation = Meadow, 0
			}
			if math.Abs(fy-74) < 1.4 || (x >= 123 && x <= 126 && y >= 66 && y <= 86) {
				t.Kind = Path
			}
		}
	}
	if RiverAt(fx, fy) {
		t.Kind = River
		if BridgeAt(fx, fy) {
			t.Kind = Bridge
		}
	}
	return t
}

// Cache shares one generated realm per seed between nodes.
type Cache struct {
	mu     sync.Mutex
	realms map[uint32]*Realm
}

func NewCache() *Cache { return &Cache{realms: map[uint32]*Realm{}} }

func (c *Cache) Get(seed uint32) *Realm {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.realms[seed]; ok {
		return r
	}
	r := New(260, 140, seed)
	c.realms[seed] = r
	return r
}

// Zone names the area at a world position for the HUD.
func Zone(x, y float64) string {
	switch {
	case RiverAt(x, y) && !BridgeAt(x, y):
		return "MIRRORSTREAM"
	case x > 84 && x < 126 && y < 76:
		return "DRAGONFALL KEEP"
	case x > 130 && y < 82 && x < 160:
		return "WYRM'S REST"
	case x > 85 && x < 133 && y > 92:
		return "MOON GROVE"
	}
	return "LOWLAND MEADOW"
}
