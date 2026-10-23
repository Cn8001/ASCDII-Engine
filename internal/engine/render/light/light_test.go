package light

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func TestUnit_FaceTowardSunIsBrighter(t *testing.T) {
	// Arrange
	env := Env{Sun: Sun{Dir: geom.V3(0, 0, 1), Color: color.RGB(255, 255, 255), Intensity: 1}, Exposure: 1}
	s := NewShader(env, nil, 0)

	// Act
	top, _, _ := s.Irradiance(geom.V3(0, 0, 1), geom.Vec3{}, 0)
	side, _, _ := s.Irradiance(geom.V3(1, 0, 0), geom.Vec3{}, 0)

	// Assert
	if !(top > side) {
		t.Fatalf("top %f side %f", top, side)
	}
}

func TestUnit_ShadowRemovesSunOnly(t *testing.T) {
	// Arrange
	env := Env{
		Sun:      Sun{Dir: geom.V3(0, 0, 1), Color: color.RGB(255, 255, 255), Intensity: 1},
		Ambient:  Ambient{Sky: color.RGB(51, 51, 51), Ground: color.RGB(51, 51, 51), Intensity: 1},
		Exposure: 1,
	}
	s := NewShader(env, nil, 0)

	// Act
	lit, _, _ := s.Irradiance(geom.V3(0, 0, 1), geom.Vec3{}, 0)
	dark, _, _ := s.Irradiance(geom.V3(0, 0, 1), geom.Vec3{}, 1)

	// Assert
	if dark < .19 || dark > .21 || lit < 1.19 {
		t.Fatalf("lit %f dark %f", lit, dark)
	}
}

func TestUnit_PointLightFallsOff(t *testing.T) {
	// Arrange
	env := Env{Exposure: 1}
	s := NewShader(env, []Point{{Pos: geom.V3(0, 0, 1), Radius: 6, Color: color.RGB(255, 128, 0), Intensity: 1}}, 0)
	up := geom.V3(0, 0, 1)

	// Act
	near, _, _ := s.Irradiance(up, geom.V3(1, 0, 0), 0)
	far, _, _ := s.Irradiance(up, geom.V3(5, 0, 0), 0)
	out, _, _ := s.Irradiance(up, geom.V3(9, 0, 0), 0)

	// Assert
	if !(near > far && far > out) || out != 0 {
		t.Fatalf("near %f far %f out %f", near, far, out)
	}
}

func TestUnit_NightIsDimmerThanNoon(t *testing.T) {
	// Arrange
	noon, _, _, _ := Daylight(12)
	night, _, _, _ := Daylight(0)

	// Act
	noonZ, nightI := noon.Dir.Z, night.Intensity

	// Assert
	if !(noon.Intensity > nightI) || noonZ < .5 {
		t.Fatalf("noon %+v night %+v", noon, night)
	}
}
