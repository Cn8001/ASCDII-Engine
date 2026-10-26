package components

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// twoParts is a model whose two faces use the materials "skin" and "eyes".
const twoParts = "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nv 2 0 0\nv 2 1 0\n" +
	"usemtl skin\nf 1 2 3 4\nusemtl eyes\nf 2 5 6 3\n"

func partsLibrary(t *testing.T) (*scene.Registry, *asset.Library) {
	t.Helper()
	reg := scene.NewRegistry()
	lib := asset.NewLibrary(reg)
	Register(reg, lib)
	m, _, err := mesh.ParseOBJ(strings.NewReader(twoParts), mesh.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lib.PutModel("models/hero.obj", m)
	lib.PutMaterial("materials/red.material.json", &material.Material{Albedo: color.RGB(200, 0, 0)})
	lib.PutMaterial("materials/green.material.json", &material.Material{Albedo: color.RGB(0, 200, 0)})
	return reg, lib
}

func TestUnit_PartSlotsFollowTheModelAndOverrideOnlyTheirPart(t *testing.T) {
	// Arrange
	_, lib := partsLibrary(t)
	m := NewMesh(lib, "")
	before := len(m.Props())

	// Act
	_ = m.Set("model", "models/hero.obj")
	_ = m.Set("material", "materials/green.material.json")
	err := m.Set(PartProp+"eyes", "materials/red.material.json")
	msh := m.mesh()
	skin, eyes := m.partMaterial(msh, msh.Parts[0]), m.partMaterial(msh, msh.Parts[1])

	// Assert
	if err != nil || len(m.Props()) != before+2 {
		t.Fatalf("err %v props %d then %d", err, before, len(m.Props()))
	}
	if eyes.Albedo != color.RGB(200, 0, 0) || skin.Albedo != color.RGB(0, 200, 0) {
		t.Fatalf("eyes %v skin %v", eyes.Albedo, skin.Albedo)
	}
	if name, ok := m.PartName(1); !ok || name != "eyes" {
		t.Fatalf("part 1 %q %v", name, ok)
	}
}

func TestUnit_PartSlotsSurviveSaveAndLoad(t *testing.T) {
	// Arrange
	reg, lib := partsLibrary(t)
	s := scene.New("t")
	o := s.Add("hero", NewMesh(lib, "models/hero.obj"))
	if err := o.Set("mesh.parts/eyes", "materials/red.material.json"); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := scene.Encode(&b, s); err != nil {
		t.Fatal(err)
	}

	// Act
	back, err := scene.Decode(&b, reg, nil)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	m, _ := back.Find("hero").Component("mesh").(*Mesh)
	if m == nil || m.slotRef("eyes") != "materials/red.material.json" || m.slotRef("skin") != "" {
		t.Fatalf("slots after load: %+v", m)
	}
}
