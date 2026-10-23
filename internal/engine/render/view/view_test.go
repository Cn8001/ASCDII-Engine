package view

import (
	"math"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func TestUnit_UnprojectInvertsProject(t *testing.T) {
	// Arrange
	c := Default()
	c.Target, c.Yaw = geom.V3(10, 20, 0), 30
	v := c.View(320, 200)
	p := geom.V3(14.5, 17.25, 2)

	// Act
	x, y, _ := v.Project(p)
	back := v.Unproject(x, y, p.Z)

	// Assert
	if back.Sub(p).Len() > 1e-9 {
		t.Fatalf("got %+v want %+v", back, p)
	}
}

func TestUnit_NearerAndHigherHaveGreaterDepth(t *testing.T) {
	// Arrange
	v := Default().View(320, 200)

	// Act
	_, _, back := v.Project(geom.V3(0, -5, 0))
	_, _, front := v.Project(geom.V3(0, 5, 0))
	_, _, top := v.Project(geom.V3(0, -5, 3))

	// Assert
	if !(front > back) || !(top > back) {
		t.Fatalf("back %f front %f top %f", back, front, top)
	}
}

func TestUnit_HeightMovesUpOnScreen(t *testing.T) {
	// Arrange
	v := Default().View(320, 200)

	// Act
	_, ground, _ := v.Project(geom.V3(0, 0, 0))
	_, raised, _ := v.Project(geom.V3(0, 0, 4))

	// Assert
	sp, cp := v.Pitch()
	if want := 4 * cp * v.Zoom; math.Abs((ground-raised)-want) > 1e-9 || sp <= 0 {
		t.Fatalf("moved %f want %f", ground-raised, want)
	}
}

func TestUnit_SpanSetsZoom(t *testing.T) {
	// Arrange
	c := Default()
	c.Span = 80

	// Act
	v := c.View(320, 200)

	// Assert
	if v.Zoom != 4 {
		t.Fatalf("zoom %f", v.Zoom)
	}
}

func TestUnit_FollowConverges(t *testing.T) {
	// Arrange
	c := Default()
	target := geom.V3(10, 0, 0)

	// Act
	for i := 0; i < 120; i++ {
		c.Follow(target, 1.0/60, 6)
	}

	// Assert
	if c.Target.Sub(target).Len() > .01 {
		t.Fatalf("got %+v", c.Target)
	}
}
