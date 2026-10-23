package anim

import (
	"math"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

type target struct {
	pos  geom.Vec3
	tint color.Color
	on   bool
}

func (t *target) bind() *prop.Binder {
	return prop.Bind().Vec3("pos", &t.pos).Color("tint", &t.tint).Bool("on", &t.on)
}
func (t *target) Props() []prop.Prop           { return t.bind().Props() }
func (t *target) Set(name string, v any) error { return t.bind().Set(name, v) }

func TestUnit_TweenInterpolatesSingleAxis(t *testing.T) {
	// Arrange
	tg := &target{}
	c := Tween("pos.z", 0.0, 2.0, 1, Linear)

	// Act
	err := c.Apply(.25, tg)

	// Assert
	if err != nil || math.Abs(tg.pos.Z-.5) > 1e-9 {
		t.Fatalf("err %v pos %+v", err, tg.pos)
	}
}

func TestUnit_PingPongReturns(t *testing.T) {
	// Arrange
	c := Tween("pos.x", 0.0, 1.0, 1, Linear).Bouncing()

	// Act
	at := c.Local(1.75)

	// Assert
	if math.Abs(at-.25) > 1e-9 {
		t.Fatalf("got %f", at)
	}
}

func TestUnit_ColorAndBoolTracks(t *testing.T) {
	// Arrange
	tg := &target{}
	c := Clip{Mode: Once, Tracks: []Track{
		{Prop: "tint", Keys: []Key{{T: 0, V: "#000000"}, {T: 2, V: "#c8c8c8"}}},
		{Prop: "on", Keys: []Key{{T: 0, V: false}, {T: 1, V: true}}},
	}}

	// Act
	err := c.Apply(1, tg)

	// Assert
	if err != nil || tg.tint != color.RGB(100, 100, 100) || !tg.on {
		t.Fatalf("err %v %+v", err, tg)
	}
}

func TestUnit_SetKeyKeepsOrderAndReplaces(t *testing.T) {
	// Arrange
	var c Clip

	// Act
	c.SetKey("pos.x", 2, 5.0, Linear)
	c.SetKey("pos.x", 0, 1.0, Linear)
	c.SetKey("pos.x", 2, 7.0, Out)

	// Assert
	k := c.Tracks[0].Keys
	if len(k) != 2 || k[0].T != 0 || k[1].V != 7.0 || k[1].Ease != Out || c.Length() != 2 {
		t.Fatalf("got %+v", k)
	}
	if !c.RemoveKey("pos.x", 0) || !c.RemoveKey("pos.x", 2) || len(c.Tracks) != 0 {
		t.Fatalf("remove failed %+v", c.Tracks)
	}
}

func TestUnit_RigRotatesAroundPivotAndInherits(t *testing.T) {
	// Arrange
	r := Rig{Bones: []Bone{{Name: "arm", Pivot: geom.V2(0, 0)}, {Name: "hand", Parent: "arm", Pivot: geom.V2(10, 0)}}}

	// Act
	xf := r.Solve(Pose{"arm": {Rot: 90}})

	// Assert
	p := xf["hand"].Apply(geom.V2(10, 0))
	if math.Abs(p.X) > 1e-9 || math.Abs(p.Y-10) > 1e-9 {
		t.Fatalf("got %+v", p)
	}
}

func TestUnit_MotionLoopsAndBlends(t *testing.T) {
	// Arrange
	m := Motion{Loop: true, Keys: []PoseKey{{T: 0, Pose: Pose{"leg": {Rot: -20}}}, {T: .5, Pose: Pose{"leg": {Rot: 20}}}, {T: 1, Pose: Pose{"leg": {Rot: -20}}}}}

	// Act
	mid, wrapped := m.Sample(.25), m.Sample(1.25)

	// Assert
	if math.Abs(mid["leg"].Rot) > 1e-9 || math.Abs(wrapped["leg"].Rot) > 1e-9 {
		t.Fatalf("mid %+v wrapped %+v", mid, wrapped)
	}
}

func TestUnit_EasesHitEndpoints(t *testing.T) {
	// Arrange
	eases := Eases()

	// Act
	for _, name := range eases {
		e := Ease(name)

		// Assert
		if math.Abs(e.Apply(0)) > 1e-9 || math.Abs(e.Apply(1)-1) > 1e-9 {
			t.Fatalf("%s endpoints %f %f", name, e.Apply(0), e.Apply(1))
		}
	}
}

func TestUnit_MoveKeyRetimesAndKeepsValue(t *testing.T) {
	// Arrange
	var c Clip
	c.SetKey("pos.x", 0, 1.0, Linear)
	c.SetKey("pos.x", 1, 2.0, Out)

	// Act
	moved := c.MoveKey("pos.x", 1, 3)
	eased := c.SetEase("pos.x", 3, Back)

	// Assert
	k := c.Tracks[0].Keys
	if !moved || !eased || len(k) != 2 || k[1].T != 3 || k[1].V != 2.0 || k[1].Ease != Back {
		t.Fatalf("keys %+v", k)
	}
	if c.MoveKey("pos.x", 9, 1) {
		t.Fatal("moved a missing key")
	}
}
