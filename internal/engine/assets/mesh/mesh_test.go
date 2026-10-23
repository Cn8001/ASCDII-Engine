package mesh

import (
	"bytes"
	"context"
	"errors"
	"image"
	stdcolor "image/color"
	"image/png"
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

const cube = `# unit cube, Y up, counter clockwise from outside
mtllib cube.mtl
v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
v 0 0 1
v 1 0 1
v 1 1 1
v 0 1 1
usemtl red
f 5 6 7 8
f 1 4 3 2
usemtl wood
f 4 8 7 3
f 1 2 6 5
f 2 3 7 6
f -8 -3 -4 -7
`

func near(a, b geom.Vec3) bool { return a.Sub(b).Len() < 1e-9 }

func TestUnit_ParseOBJTriangulatesQuads(t *testing.T) {
	// Arrange
	r := strings.NewReader(cube)

	// Act
	m, libs, err := ParseOBJ(r, Options{})

	// Assert
	if err != nil || len(m.Tris) != 12 || len(m.Parts) != 2 || len(libs) != 1 || libs[0] != "cube.mtl" {
		t.Fatalf("err %v tris %d parts %d libs %v", err, len(m.Tris), len(m.Parts), libs)
	}
	if !near(m.Bounds.Min, geom.V3(0, 0, 0)) || !near(m.Bounds.Max, geom.V3(1, 1, 1)) {
		t.Fatalf("bounds %+v", m.Bounds)
	}
}

func TestUnit_YUpBecomesZUpWithOutwardNormals(t *testing.T) {
	// Arrange
	m, _, err := ParseOBJ(strings.NewReader(cube), Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	front := m.Normal(m.Tris[0]) // OBJ +Z face, towards an OBJ viewer
	top := m.Normal(m.Tris[4])   // OBJ +Y face

	// Assert
	if !near(front, geom.V3(0, 1, 0)) || !near(top, geom.V3(0, 0, 1)) {
		t.Fatalf("front %+v top %+v", front, top)
	}
}

func TestUnit_ScaleAndZUpOption(t *testing.T) {
	// Arrange
	src := "v 1 2 3\nv 0 0 0\nv 0 0 1\nf 1 2 3\n"

	// Act
	m, _, err := ParseOBJ(strings.NewReader(src), Options{UpZ: true, Scale: 2})

	// Assert
	if err != nil || !near(m.Pos[0], geom.V3(2, -4, 6)) {
		t.Fatalf("err %v pos %+v", err, m.Pos)
	}
}

func TestUnit_BadFaceIndexFails(t *testing.T) {
	// Arrange
	src := "v 0 0 0\nv 1 0 0\nf 1 2 9\n"

	// Act
	_, _, err := ParseOBJ(strings.NewReader(src), Options{})

	// Assert
	if !errors.Is(err, ErrBadOBJ) {
		t.Fatalf("err %v", err)
	}
}

func png2x2(c stdcolor.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < 4; i++ {
		img.Set(i%2, i/2, c)
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestUnit_LoadAppliesMaterialsAndTextureColor(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{
		"models/cube.obj":     {Data: []byte(cube)},
		"models/cube.mtl":     {Data: []byte("newmtl red\nKd 1 0 0\nnewmtl wood\nKd 1 1 1\nmap_Kd -s 1 1 1 tex\\wood.png\n")},
		"models/tex/wood.png": {Data: png2x2(stdcolor.RGBA{R: 100, G: 60, B: 20, A: 255})},
	}

	// Act
	m, err := Load(context.Background(), fsys, "models/cube.obj", Options{})
	files, errFiles := Files(fsys, "models/cube.obj")

	// Assert
	if err != nil || errFiles != nil {
		t.Fatalf("err %v %v", err, errFiles)
	}
	wood := m.Parts[1].Color
	if m.Parts[0].Color != color.RGB(255, 0, 0) || math.Abs(float64(wood.R)-100) > 1 || m.Parts[1].Texture != "tex/wood.png" {
		t.Fatalf("parts %+v", m.Parts)
	}
	if len(files) != 2 || files[0] != "cube.mtl" || files[1] != "tex/wood.png" {
		t.Fatalf("files %v", files)
	}
}

func TestUnit_TextureCoordinatesFollowCorners(t *testing.T) {
	// Arrange
	src := "v 0 0 0\nv 1 0 0\nv 1 1 0\nvt 0 0\nvt 1 0\nvt 1 1\nf 1/1 2/2 3/3\nf 1 2 3\n"

	// Act
	m, _, err := ParseOBJ(strings.NewReader(src), Options{})

	// Assert
	if err != nil || len(m.UV) != 3 || !m.Tris[0].HasUV() || m.Tris[1].HasUV() {
		t.Fatalf("err %v tris %+v", err, m.Tris)
	}
	if m.UV[m.Tris[0].T[1]] != geom.V2(1, 1) {
		t.Fatalf("winding swap lost the uv pairing: %+v", m.Tris[0])
	}
}
