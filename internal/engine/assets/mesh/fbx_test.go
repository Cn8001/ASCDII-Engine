package mesh

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	stdcolor "image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/color"
	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/geom"
)

// fbxCube is the OBJ test cube as an ASCII FBX 7.4 file: Y up, a parent
// Null holding the cube, a red and a textured wood material.
type fbxCube struct {
	global string // extra GlobalSettings P lines
	cube   string // extra P lines on the cube model
	parent string // extra P lines on the parent
	unit   string
}

func (c fbxCube) String() string {
	unit := c.unit
	if unit == "" {
		unit = "100"
	}
	return `; FBX 7.4.0 project file
FBXHeaderExtension:  {
	FBXHeaderVersion: 1003
	FBXVersion: 7400
}
GlobalSettings:  {
	Version: 1000
	Properties70:  {
		P: "UnitScaleFactor", "double", "Number", "",` + unit + `
` + c.global + `
	}
}
Objects:  {
	Geometry: 10, "Geometry::Cube", "Mesh" {
		Vertices: *24 {
			a: 0,0,0,1,0,0,1,1,0,0,1,0,0,0,1,1,0,1,1,1,1,0,1,1
		}
		PolygonVertexIndex: *24 {
			a: 4,5,6,-8,0,3,2,-2,3,7,6,-3,0,1,5,-5,1,2,6,-6,0,4,7,-4
		}
		LayerElementUV: 0 {
			MappingInformationType: "ByPolygonVertex"
			ReferenceInformationType: "IndexToDirect"
			UV: *8 {
				a: 0,0,1,0,1,1,0,1
			}
			UVIndex: *24 {
				a: 0,1,2,3,0,1,2,3,0,1,2,3,0,1,2,3,0,1,2,3,0,1,2,3
			}
		}
		LayerElementMaterial: 0 {
			MappingInformationType: "ByPolygon"
			ReferenceInformationType: "IndexToDirect"
			Materials: *6 {
				a: 0,0,1,1,1,1
			}
		}
	}
	Model: 21, "Model::Root", "Null" {
		Properties70:  {
` + c.parent + `
		}
	}
	Model: 20, "Model::Cube", "Mesh" {
		Properties70:  {
` + c.cube + `
		}
		Shading: T
	}
	Material: 30, "Material::red", "" {
		Properties70:  {
			P: "DiffuseColor", "Color", "", "A",1,0,0
		}
	}
	Material: 31, "Material::wood", "" {
		Properties70:  {
			P: "DiffuseColor", "Color", "", "A",1,1,1
		}
	}
	Texture: 40, "Texture::wood", "" {
		FileName: "C:\art\wood.png"
		RelativeFilename: "tex\wood.png"
	}
}
Connections:  {
	C: "OO",21,0
	C: "OO",20,21
	C: "OO",10,20
	C: "OO",30,20
	C: "OO",31,20
	C: "OP",40,31, "DiffuseColor"
}
`
}

// encodeFBX writes nodes as a binary FBX 7.4 file, arrays deflated.
func encodeFBX(t *testing.T, roots []*fbxNode) []byte {
	t.Helper()
	le := binary.LittleEndian
	var b bytes.Buffer
	b.WriteString(fbxMagic + "\x1a\x00")
	b.Write(le.AppendUint32(nil, 7400))
	var node func(n *fbxNode)
	node = func(n *fbxNode) {
		start := b.Len()
		b.Write(make([]byte, 12))
		b.WriteByte(byte(len(n.Name)))
		b.WriteString(n.Name)
		props := b.Len()
		for _, p := range n.Props {
			switch v := p.(type) {
			case int64:
				b.WriteByte('L')
				b.Write(le.AppendUint64(nil, uint64(v)))
			case float64:
				b.WriteByte('D')
				b.Write(le.AppendUint64(nil, math.Float64bits(v)))
			case string:
				b.WriteByte('S')
				b.Write(le.AppendUint32(nil, uint32(len(v))))
				b.WriteString(v)
			case []int64:
				var raw []byte
				for _, x := range v {
					raw = le.AppendUint32(raw, uint32(int32(x)))
				}
				array(t, &b, 'i', len(v), raw)
			case []float64:
				var raw []byte
				for _, x := range v {
					raw = le.AppendUint64(raw, math.Float64bits(x))
				}
				array(t, &b, 'd', len(v), raw)
			default:
				t.Fatalf("encode %T", p)
			}
		}
		plen := b.Len() - props
		for _, k := range n.Kids {
			node(k)
		}
		if len(n.Kids) > 0 {
			b.Write(make([]byte, 13))
		}
		raw := b.Bytes()
		le.PutUint32(raw[start:], uint32(b.Len()))
		le.PutUint32(raw[start+4:], uint32(len(n.Props)))
		le.PutUint32(raw[start+8:], uint32(plen))
	}
	for _, n := range roots {
		node(n)
	}
	b.Write(make([]byte, 13))
	return b.Bytes()
}

func array(t *testing.T, b *bytes.Buffer, kind byte, n int, raw []byte) {
	t.Helper()
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	if _, err := w.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	b.WriteByte(kind)
	b.Write(le.AppendUint32(nil, uint32(n)))
	b.Write(le.AppendUint32(nil, 1))
	b.Write(le.AppendUint32(nil, uint32(z.Len())))
	b.Write(z.Bytes())
}

func binaryCube(t *testing.T, c fbxCube) []byte {
	t.Helper()
	roots, _, err := readFBX([]byte(c.String()))
	if err != nil {
		t.Fatal(err)
	}
	return encodeFBX(t, roots)
}

// outward reports whether every triangle of a closed mesh faces away from
// its center.
func outward(m *Mesh) bool {
	mid := m.Bounds.Min.Lerp(m.Bounds.Max, .5)
	for _, tr := range m.Tris {
		c := m.Pos[tr.V[0]].Add(m.Pos[tr.V[1]]).Add(m.Pos[tr.V[2]]).Scale(1.0 / 3)
		if m.Normal(tr).Dot(c.Sub(mid)) <= 0 {
			return false
		}
	}
	return true
}

func TestUnit_FBXCubeReadsLikeTheOBJCube(t *testing.T) {
	// Arrange
	src := []byte(fbxCube{}.String())

	// Act
	m, texs, err := parseFBX(src, Options{})

	// Assert
	if err != nil || len(m.Tris) != 12 || len(m.Parts) != 2 {
		t.Fatalf("err %v mesh %+v", err, m)
	}
	if m.Parts[0].Material != "red" || m.Parts[0].Color != color.RGB(255, 0, 0) || m.Parts[1].Material != "wood" {
		t.Fatalf("parts %+v", m.Parts)
	}
	if !reflect.DeepEqual(texs, [][]string{nil, {"tex/wood.png", "wood.png"}}) {
		t.Fatalf("textures %v", texs)
	}
	if !near(m.Bounds.Min, geom.V3(0, 0, 0)) || !near(m.Bounds.Max, geom.V3(1, 1, 1)) {
		t.Fatalf("bounds %+v", m.Bounds)
	}
	if !near(m.Normal(m.Tris[0]), geom.V3(0, 1, 0)) || !near(m.Normal(m.Tris[4]), geom.V3(0, 0, 1)) || !outward(m) {
		t.Fatalf("front %+v top %+v", m.Normal(m.Tris[0]), m.Normal(m.Tris[4]))
	}
}

func TestUnit_FBXBinaryMatchesASCII(t *testing.T) {
	// Arrange
	want, _, err := parseFBX([]byte(fbxCube{}.String()), Options{})
	if err != nil {
		t.Fatal(err)
	}
	bin := binaryCube(t, fbxCube{})

	// Act
	got, _, err := parseFBX(bin, Options{})

	// Assert
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("err %v\ngot  %+v\nwant %+v", err, got, want)
	}
}

func TestUnit_FBXCentimetersBecomeMeters(t *testing.T) {
	// Arrange
	src := []byte(fbxCube{unit: "1"}.String())

	// Act
	m, _, err := parseFBX(src, Options{Scale: 2})

	// Assert
	if err != nil || !near(m.Bounds.Max, geom.V3(.02, .02, .02)) {
		t.Fatalf("err %v bounds %+v", err, m.Bounds)
	}
}

func TestUnit_FBXParentTransformApplies(t *testing.T) {
	// Arrange
	src := []byte(fbxCube{parent: `P: "Lcl Translation", "Lcl Translation", "", "A",0,2,0
P: "Lcl Scaling", "Lcl Scaling", "", "A",2,2,2`}.String())

	// Act
	m, _, err := parseFBX(src, Options{})

	// Assert
	if err != nil || !near(m.Bounds.Min, geom.V3(0, 0, 2)) || !near(m.Bounds.Max, geom.V3(2, 2, 4)) {
		t.Fatalf("err %v bounds %+v", err, m.Bounds)
	}
}

func TestUnit_FBXRotationAndMirrorKeepFacesOutward(t *testing.T) {
	cases := map[string]struct {
		cube     string
		min, max geom.Vec3
	}{
		"rotated":  {`P: "Lcl Rotation", "Lcl Rotation", "", "A",0,0,90`, geom.V3(-1, 0, 0), geom.V3(0, 1, 1)},
		"mirrored": {`P: "Lcl Scaling", "Lcl Scaling", "", "A",-1,1,1`, geom.V3(-1, 0, 0), geom.V3(0, 1, 1)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			src := []byte(fbxCube{cube: tc.cube}.String())

			// Act
			m, _, err := parseFBX(src, Options{})

			// Assert
			if err != nil || !near(m.Bounds.Min, tc.min) || !near(m.Bounds.Max, tc.max) || !outward(m) {
				t.Fatalf("err %v bounds %+v outward %v", err, m.Bounds, outward(m))
			}
		})
	}
}

func TestUnit_FBXZUpAxesMatchOBJUpZ(t *testing.T) {
	// Arrange
	src := []byte(fbxCube{global: `P: "UpAxis", "int", "Integer", "",2
P: "UpAxisSign", "int", "Integer", "",1
P: "FrontAxis", "int", "Integer", "",1
P: "FrontAxisSign", "int", "Integer", "",-1
P: "CoordAxis", "int", "Integer", "",0
P: "CoordAxisSign", "int", "Integer", "",1`}.String())
	objUpZ, _, err := ParseOBJ(strings.NewReader(cube), Options{UpZ: true})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	m, _, err := parseFBX(src, Options{UpZ: true})

	// Assert
	if err != nil || !near(m.Bounds.Min, objUpZ.Bounds.Min) || !near(m.Bounds.Max, objUpZ.Bounds.Max) || !outward(m) {
		t.Fatalf("err %v bounds %+v want %+v", err, m.Bounds, objUpZ.Bounds)
	}
}

func TestUnit_FBXTextureCoordinatesFollowCorners(t *testing.T) {
	// Arrange
	src := []byte(fbxCube{}.String())

	// Act
	m, _, err := parseFBX(src, Options{})

	// Assert
	if err != nil || len(m.UV) != 4 || !m.Tris[0].HasUV() {
		t.Fatalf("err %v tris %+v", err, m.Tris)
	}
	if m.UV[m.Tris[0].T[1]] != geom.V2(1, 1) {
		t.Fatalf("winding swap lost the uv pairing: %+v", m.Tris[0])
	}
}

func TestUnit_LoadFBXFindsTexturesAndLists(t *testing.T) {
	cases := map[string]string{"relative path": "tex/wood.png", "next to the model": "wood.png"}
	for name, tex := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			fsys := fstest.MapFS{
				"models/cube.fbx":  {Data: binaryCube(t, fbxCube{})},
				"models/" + tex:    {Data: png2x2(stdcolor.RGBA{R: 100, G: 60, B: 20, A: 255})},
				"models/other.png": {Data: png2x2(stdcolor.RGBA{A: 255})},
			}

			// Act
			m, err := Load(context.Background(), fsys, "models/cube.fbx", Options{})
			files, errFiles := Files(fsys, "models/cube.fbx")

			// Assert
			if err != nil || errFiles != nil {
				t.Fatalf("err %v %v", err, errFiles)
			}
			if m.Parts[1].Texture != tex || math.Abs(float64(m.Parts[1].Color.R)-100) > 1 || m.Parts[0].Texture != "" {
				t.Fatalf("parts %+v", m.Parts)
			}
			if !reflect.DeepEqual(files, []string{tex}) {
				t.Fatalf("files %v", files)
			}
		})
	}
}

func TestUnit_LoadFBXWithoutTextureDrawsInColor(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{"cube.FBX": {Data: []byte(fbxCube{}.String())}}

	// Act
	m, err := Load(context.Background(), fsys, "cube.FBX", Options{})

	// Assert
	if err != nil || m.Parts[1].Texture != "" || m.Parts[1].Color != color.RGB(255, 255, 255) {
		t.Fatalf("err %v parts %+v", err, m.Parts)
	}
}

func TestUnit_FBXRejectsOldAndBrokenFiles(t *testing.T) {
	cases := map[string][]byte{
		"fbx 6":     []byte(strings.Replace(fbxCube{}.String(), "FBXVersion: 7400", "FBXVersion: 6100", 1)),
		"not fbx":   []byte("hello"),
		"no faces":  []byte("; FBX 7.4.0 project file\nFBXHeaderExtension: {\n}\nObjects: {\n}\n"),
		"bad index": []byte(strings.Replace(fbxCube{}.String(), "4,5,6,-8", "4,5,60,-8", 1)),
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, _, err := parseFBX(src, Options{})

			// Assert
			if !errors.Is(err, ErrBadFBX) {
				t.Fatalf("err %v", err)
			}
		})
	}
}

func TestUnit_FBXTruncatedBinaryFailsWithoutPanic(t *testing.T) {
	// Arrange
	bin := binaryCube(t, fbxCube{})

	for n := range len(bin) - 13 {
		// Act
		_, _, err := parseFBX(bin[:n], Options{})

		// Assert
		if err == nil {
			t.Fatalf("prefix of %d bytes parsed", n)
		}
	}
}
