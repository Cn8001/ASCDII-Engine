package physics

import (
	"math"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func near(a, b geom.Vec2) bool { return a.Sub(b).Len() < 1e-6 }

func TestUnit_OverlapPushesOutAlongShortestAxis(t *testing.T) {
	// Arrange
	box := Box(geom.V2(0, 0), geom.V2(1, 1), 0)

	// Act
	cc, okCC := Overlap(Circle{C: geom.V2(1.5, 0), R: 1}, Circle{C: geom.V2(0, 0), R: 1})
	cp, okCP := Overlap(Circle{C: geom.V2(1.5, 0), R: 1}, box)
	pp, okPP := Overlap(Box(geom.V2(0, 1.8), geom.V2(1, 1), 0), box)
	_, apart := Overlap(Circle{C: geom.V2(5, 0), R: 1}, box)

	// Assert
	if !okCC || !near(cc, geom.V2(.5, 0)) || !okCP || !near(cp, geom.V2(.5, 0)) || !okPP || !near(pp, geom.V2(0, .2)) || apart {
		t.Fatalf("cc %v cp %v pp %v apart %v", cc, cp, pp, apart)
	}
}

func TestUnit_RotatedBoxAndHull(t *testing.T) {
	// Arrange
	diamond := Box(geom.V2(0, 0), geom.V2(1, 1), math.Pi/4)
	hull := Hull([]geom.Vec2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2}, {X: 0, Y: 2}})

	// Act
	inCorner := Contains(diamond, geom.V2(.9, .9))
	onTip := Contains(diamond, geom.V2(1.3, 0))
	inside := Contains(hull, geom.V2(1, 1.5))

	// Assert
	if inCorner || !onTip || !inside || len(hull.Pts) != 4 {
		t.Fatalf("corner %v tip %v inside %v hull %v", inCorner, onTip, inside, hull.Pts)
	}
}
