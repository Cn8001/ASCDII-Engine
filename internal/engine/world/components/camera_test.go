package components

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func TestUnit_CameraObjectDrivesTheViewAndMovesWithItsParent(t *testing.T) {
	// Arrange
	s := scene.New("t")
	s.Camera = view.Camera{Target: geom.V3(1, 1, 0), Span: 99}
	hero := s.Add("hero", nil).At(10, 5, 0)
	cam := NewCamera()
	cam.Span = 20
	o := hero.Add("cam", cam).At(0, 2, 0)
	o.Local.Rot.Z = 30

	// Act
	before := s.View()
	hero.Local.Pos = geom.V3(20, 5, 0)
	after := s.View()
	o.Visible = false
	fallback := s.View()

	// Assert
	if before.Target != geom.V3(10, 7, 0) || before.Span != 20 || before.Yaw != 30 || after.Target != geom.V3(20, 7, 0) || fallback.Span != 99 {
		t.Fatalf("before %+v after %+v fallback %+v", before, after, fallback)
	}
}

func TestUnit_SetViewPlacesTheCameraObject(t *testing.T) {
	// Arrange
	s := scene.New("t")
	cam := NewCamera()
	o := s.Add("cam", cam)

	// Act
	s.SetView(view.Camera{Target: geom.V3(4, 5, 0), Span: 30, Pitch: 40, Yaw: 15, AnchorX: .5, AnchorY: .5})
	errSet := s.CameraProps().Set("span", 12.0)

	// Assert
	if errSet != nil || o.Local.Pos != geom.V3(4, 5, 0) || o.Local.Rot.Z != 15 || cam.Span != 12 || cam.Pitch != 40 {
		t.Fatalf("err %v pos %+v rot %+v cam %+v", errSet, o.Local.Pos, o.Local.Rot, cam)
	}
}

func TestUnit_FollowLeavesACameraParentedToTheFollower(t *testing.T) {
	// Arrange
	s := scene.New("t")
	hero := s.Add("hero", nil).At(10, 0, 0)
	o := hero.Add("cam", NewCamera()).At(0, 3, 0)
	loose := scene.New("u")
	free := loose.Add("cam", NewCamera())

	// Act
	s.FollowView(hero, geom.V3(10, 0, 0), 1, 5)
	loose.FollowView(nil, geom.V3(10, 0, 0), 1, 5)

	// Assert
	if o.Local.Pos != geom.V3(0, 3, 0) || free.Local.Pos.X <= 0 {
		t.Fatalf("child %+v free %+v", o.Local.Pos, free.Local.Pos)
	}
}
