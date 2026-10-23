package draw

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

func lit(b *raster.Buffer) (n int) {
	for _, d := range b.Dots() {
		if d.Flags&raster.FlagEmpty == 0 {
			n++
		}
	}
	return n
}

func TestUnit_SmoothSpriteKeepsCoveredDotsAndDropsSlivers(t *testing.T) {
	// Arrange
	c := newContext(40, 20)
	scale := .25 / c.View.Zoom // four art pixels per dot, so Smooth applies

	// Act
	s := c.Sprite(geom.V3(0, 0, 0), scale, false).Smooth(40)
	s.Color(M(color.RGB(200, 40, 40))).Rect(-16, -32, 32, 32)
	s.Color(M(color.RGB(40, 200, 40))).Rect(20, -32, 1, 1)
	s.End()
	block := lit(c.Buf)
	second := c.Sprite(geom.V3(0, 0, 0), scale, false).Smooth(40)
	second.Color(M(color.RGB(40, 40, 200))).Rect(30, -10, 1, 1)
	second.End()

	// Assert
	if block < 60 || block > 72 || lit(c.Buf) != block {
		t.Fatalf("block %d dots, after sliver %d", block, lit(c.Buf))
	}
}

func TestUnit_SmoothIsANoOpAtFullResolution(t *testing.T) {
	// Arrange
	c := newContext(40, 20)

	// Act
	s := c.Sprite(geom.V3(0, 0, 0), 1/c.View.Zoom, false).Smooth(40)

	// Assert
	if s.ss != nil {
		t.Fatal("a sprite at one dot per art pixel was supersampled")
	}
}
