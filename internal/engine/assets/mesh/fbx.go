package mesh

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"path"
	"slices"
	"strings"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/color"
	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/geom"
)

func isFBX(p string) bool { return strings.EqualFold(path.Ext(p), ".fbx") }

type mat4 [4][4]float64

func ident() mat4 {
	var m mat4
	for i := range 4 {
		m[i][i] = 1
	}
	return m
}

func (a mat4) mul(b mat4) mat4 {
	var m mat4
	for r := range 4 {
		for c := range 4 {
			for k := range 4 {
				m[r][c] += a[r][k] * b[k][c]
			}
		}
	}
	return m
}

func (a mat4) point(p geom.Vec3) geom.Vec3 {
	return geom.V3(
		a[0][0]*p.X+a[0][1]*p.Y+a[0][2]*p.Z+a[0][3],
		a[1][0]*p.X+a[1][1]*p.Y+a[1][2]*p.Z+a[1][3],
		a[2][0]*p.X+a[2][1]*p.Y+a[2][2]*p.Z+a[2][3],
	)
}

func (a mat4) det3() float64 {
	return a[0][0]*(a[1][1]*a[2][2]-a[1][2]*a[2][1]) -
		a[0][1]*(a[1][0]*a[2][2]-a[1][2]*a[2][0]) +
		a[0][2]*(a[1][0]*a[2][1]-a[1][1]*a[2][0])
}

// transposed inverts a pure rotation.
func (a mat4) transposed() mat4 {
	m := ident()
	for r := range 3 {
		for c := range 3 {
			m[r][c] = a[c][r]
		}
	}
	return m
}

func translation(v geom.Vec3) mat4 {
	m := ident()
	m[0][3], m[1][3], m[2][3] = v.X, v.Y, v.Z
	return m
}

func scaling(v geom.Vec3) mat4 {
	m := ident()
	m[0][0], m[1][1], m[2][2] = v.X, v.Y, v.Z
	return m
}

func axisRotation(axis int, deg float64) mat4 {
	s, c := math.Sincos(deg * math.Pi / 180)
	m := ident()
	i, j := (axis+1)%3, (axis+2)%3
	m[i][i], m[i][j], m[j][i], m[j][j] = c, -s, s, c
	return m
}

// euler composes rotations in FBX order: 0 XYZ applies X first, then Y,
// then Z; 1 XZY, 2 YZX, 3 YXZ, 4 ZXY, 5 ZYX. Spheric XYZ (6) reads as XYZ.
func euler(deg geom.Vec3, order int) mat4 {
	orders := [6][3]int{{0, 1, 2}, {0, 2, 1}, {1, 2, 0}, {1, 0, 2}, {2, 0, 1}, {2, 1, 0}}
	if order < 0 || order >= len(orders) {
		order = 0
	}
	a := [3]float64{deg.X, deg.Y, deg.Z}
	m := ident()
	for _, ax := range orders[order] {
		m = axisRotation(ax, a[ax]).mul(m)
	}
	return m
}

type fbxProps map[string]*fbxNode

// props70 indexes a node's Properties70 by name; each P holds name, type,
// label, flags and then the values.
func props70(n *fbxNode) fbxProps {
	out := fbxProps{}
	for _, p := range n.kid("Properties70").kids() {
		if p.Name == "P" {
			out[p.str(0)] = p
		}
	}
	return out
}

func (ps fbxProps) num(name string, def float64) float64 {
	if v, ok := ps[name].num(4); ok {
		return v
	}
	return def
}

func (ps fbxProps) vec(name string, def geom.Vec3) geom.Vec3 {
	p := ps[name]
	x, okx := p.num(4)
	y, oky := p.num(5)
	z, okz := p.num(6)
	if !okx || !oky || !okz {
		return def
	}
	return geom.V3(x, y, z)
}

// local is the FBX node transform:
// T * Roff * Rp * Rpre * R * Rpost^-1 * Rp^-1 * Soff * Sp * S * Sp^-1.
func (ps fbxProps) local() mat4 {
	zero, one := geom.Vec3{}, geom.V3(1, 1, 1)
	rp, sp := ps.vec("RotationPivot", zero), ps.vec("ScalingPivot", zero)
	return translation(ps.vec("Lcl Translation", zero)).
		mul(translation(ps.vec("RotationOffset", zero))).
		mul(translation(rp)).
		mul(euler(ps.vec("PreRotation", zero), 0)).
		mul(euler(ps.vec("Lcl Rotation", zero), int(ps.num("RotationOrder", 0)))).
		mul(euler(ps.vec("PostRotation", zero), 0).transposed()).
		mul(translation(rp.Scale(-1))).
		mul(translation(ps.vec("ScalingOffset", zero))).
		mul(translation(sp)).
		mul(scaling(ps.vec("Lcl Scaling", one))).
		mul(translation(sp.Scale(-1)))
}

// geometric is the offset a 3ds Max pivot leaves on the mesh only; children
// do not inherit it.
func (ps fbxProps) geometric() mat4 {
	return translation(ps.vec("GeometricTranslation", geom.Vec3{})).
		mul(euler(ps.vec("GeometricRotation", geom.Vec3{}), 0)).
		mul(scaling(ps.vec("GeometricScaling", geom.V3(1, 1, 1))))
}

// canonical maps the file's declared axes onto Y up, Z towards the viewer,
// in meters, which is what an OBJ export holds, so Options.engine applies.
func (ps fbxProps) canonical() mat4 {
	up, front, coord := int(ps.num("UpAxis", 1)), int(ps.num("FrontAxis", 2)), int(ps.num("CoordAxis", 0))
	sign := func(name string) float64 {
		if ps.num(name, 1) < 0 {
			return -1
		}
		return 1
	}
	m := ident()
	if valid := up != front && up != coord && front != coord && min(up, front, coord) >= 0 && max(up, front, coord) <= 2; valid {
		m = mat4{}
		m[0][coord], m[1][up], m[2][front], m[3][3] = sign("CoordAxisSign"), sign("UpAxisSign"), sign("FrontAxisSign"), 1
	}
	cm := ps.num("UnitScaleFactor", 1)
	if cm <= 0 {
		cm = 1
	}
	return scaling(geom.V3(1, 1, 1).Scale(cm / 100)).mul(m)
}

type fbxScene struct {
	objs   map[int64]*fbxNode
	order  []int64
	kids   map[int64][]int64            // object to object, in file order
	parent map[int64]int64              // model to parent model
	slots  map[int64]map[string][]int64 // object to property, such as a texture on DiffuseColor
	world  map[int64]mat4
}

func newFBXScene(objects, connections *fbxNode) *fbxScene {
	s := &fbxScene{objs: map[int64]*fbxNode{}, kids: map[int64][]int64{}, parent: map[int64]int64{}, slots: map[int64]map[string][]int64{}, world: map[int64]mat4{}}
	for _, n := range objects.kids() {
		id := n.i64(0)
		s.objs[id] = n
		s.order = append(s.order, id)
	}
	for _, c := range connections.kids() {
		if c.Name != "C" {
			continue
		}
		child, parent := c.i64(1), c.i64(2)
		switch c.str(0) {
		case "OO":
			s.kids[parent] = append(s.kids[parent], child)
			if s.objs[child].is("Model") && s.objs[parent].is("Model") {
				s.parent[child] = parent
			}
		case "OP":
			if s.slots[parent] == nil {
				s.slots[parent] = map[string][]int64{}
			}
			s.slots[parent][c.str(3)] = append(s.slots[parent][c.str(3)], child)
		}
	}
	return s
}

func (s *fbxScene) worldOf(id int64, depth int) mat4 {
	if w, ok := s.world[id]; ok {
		return w
	}
	w := props70(s.objs[id]).local()
	if p, ok := s.parent[id]; ok && depth < 64 {
		w = s.worldOf(p, depth+1).mul(w)
	}
	s.world[id] = w
	return w
}

func (s *fbxScene) children(id int64, name string) []*fbxNode {
	var out []*fbxNode
	for _, k := range s.kids[id] {
		if n := s.objs[k]; n.is(name) {
			out = append(out, n)
		}
	}
	return out
}

// diffuseTexture finds the texture on a material's diffuse or base color.
func (s *fbxScene) diffuseTexture(mat int64) *fbxNode {
	slots := s.slots[mat]
	names := make([]string, 0, len(slots))
	for k := range slots {
		names = append(names, k)
	}
	rank := func(k string) int {
		l := strings.ToLower(k)
		switch {
		case k == "DiffuseColor":
			return 0
		case strings.Contains(l, "diffuse"):
			return 1
		case strings.Contains(l, "basecolor"), strings.Contains(l, "base_color"):
			return 2
		}
		return -1
	}
	slices.SortFunc(names, func(a, b string) int {
		if d := rank(a) - rank(b); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	for _, k := range names {
		if rank(k) < 0 {
			continue
		}
		for _, id := range slots[k] {
			if t := s.objs[id]; t.is("Texture") {
				return t
			}
		}
	}
	return nil
}

// fbxName strips the class from an object name, "Cube\x00\x01Model" in
// binary files and "Model::Cube" in ASCII ones.
func fbxName(s string) string {
	if a, _, ok := strings.Cut(s, "\x00\x01"); ok {
		return a
	}
	if _, b, ok := strings.Cut(s, "::"); ok {
		return b
	}
	return s
}

// texturePaths lists where a texture may be relative to the model, best
// first. Exporters write absolute paths of the artist's machine, so the bare
// file name next to the model is also tried.
func texturePaths(t *fbxNode) []string {
	slash := func(s string) string { return strings.ReplaceAll(s, "\\", "/") }
	rel, abs := slash(t.kid("RelativeFilename").str(0)), slash(t.kid("FileName").str(0))
	var out []string
	for _, c := range []string{rel, path.Base(rel), path.Base(abs)} {
		c = path.Clean(c)
		if c == "." || c == "/" || path.IsAbs(c) || strings.HasPrefix(c, "..") || strings.Contains(c, ":") || slices.Contains(out, c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

type fbxBuild struct {
	m     *Mesh
	s     *fbxScene
	parts map[int64]int
	texs  [][]string
}

func (b *fbxBuild) part(mat *fbxNode) int {
	id := mat.i64(0)
	if i, ok := b.parts[id]; ok {
		return i
	}
	p := Part{Color: clay(), Kd: clay()}
	var tex []string
	if mat != nil {
		ps := props70(mat)
		p.Material = fbxName(mat.str(1))
		if c := ps.vec("DiffuseColor", ps.vec("Diffuse", geom.V3(-1, 0, 0))); c.X >= 0 {
			p.Color = color.FromFloat(c.X, c.Y, c.Z)
			p.Kd = p.Color
		}
		if t := b.s.diffuseTexture(id); t != nil {
			tex = texturePaths(t)
		}
	}
	b.m.Parts = append(b.m.Parts, p)
	b.texs = append(b.texs, tex)
	b.parts[id] = len(b.m.Parts) - 1
	return b.parts[id]
}

// layer reads a LayerElement's lookup: which entry a polygon corner, its
// control point or its polygon uses, -1 when the layer has none.
func layer(n *fbxNode, index string) func(corner, vertex, poly int) int {
	if n == nil {
		return func(int, int, int) int { return -1 }
	}
	idx := n.kid(index).ints()
	mapping := n.kid("MappingInformationType").str(0)
	indexed := strings.HasPrefix(n.kid("ReferenceInformationType").str(0), "Index")
	return func(corner, vertex, poly int) int {
		i := corner
		switch mapping {
		case "ByVertice", "ByVertex", "ByControlPoint":
			i = vertex
		case "ByPolygon":
			i = poly
		case "AllSame":
			i = 0
		}
		if indexed {
			if i < 0 || i >= len(idx) {
				return -1
			}
			i = int(idx[i])
		}
		return i
	}
}

func (b *fbxBuild) geometry(g *fbxNode, xf mat4, mats []*fbxNode, o Options) error {
	verts := g.kid("Vertices").floats()
	polys := g.kid("PolygonVertexIndex").ints()
	base := len(b.m.Pos)
	for i := 0; i+2 < len(verts); i += 3 {
		b.m.Pos = append(b.m.Pos, o.engine(xf.point(geom.V3(verts[i], verts[i+1], verts[i+2]))))
	}
	nv := len(verts) / 3

	uvNode := g.kid("LayerElementUV")
	uvAt := layer(uvNode, "UVIndex")
	uvs := uvNode.kid("UV").floats()
	uvBase := len(b.m.UV)
	for i := 0; i+1 < len(uvs); i += 2 {
		b.m.UV = append(b.m.UV, geom.V2(uvs[i], uvs[i+1]))
	}
	matNode := g.kid("LayerElementMaterial")
	mi, perPoly := matNode.kid("Materials").ints(), matNode.kid("MappingInformationType").str(0) == "ByPolygon"
	matAt := func(poly int) int {
		switch {
		case len(mi) == 0:
			return 0
		case perPoly && poly < len(mi):
			return int(mi[poly])
		}
		return int(mi[0])
	}

	k := o.Scale
	if k == 0 {
		k = 1
	}
	swap := xf.det3()*k > 0 // Options.engine mirrors, so a non-mirrored node swaps
	var vs, ts []int
	poly := 0
	for c, raw := range polys {
		v := int(raw)
		end := raw < 0
		if end {
			v = ^v
		}
		if v >= nv {
			return fmt.Errorf("%w: %s: vertex %d out of range", ErrBadFBX, fbxName(g.str(1)), v)
		}
		t := uvAt(c, v, poly)
		if t < 0 || t >= len(uvs)/2 {
			t = -1
		} else {
			t += uvBase
		}
		vs, ts = append(vs, base+v), append(ts, t)
		if !end {
			continue
		}
		var mat *fbxNode
		if mi := matAt(poly); mi >= 0 && mi < len(mats) {
			mat = mats[mi]
		}
		part := b.part(mat)
		for j := 2; j < len(vs); j++ {
			a, z := j-1, j
			if swap {
				a, z = j, j-1
			}
			b.m.Tris = append(b.m.Tris, Tri{V: [3]int{vs[0], vs[a], vs[z]}, T: [3]int{ts[0], ts[a], ts[z]}, Part: part})
		}
		vs, ts = vs[:0], ts[:0]
		poly++
	}
	return nil
}

// parseFBX reads the static meshes of an FBX 7 file, binary or ASCII, with
// every node transform applied. It also returns, per part, the places its
// diffuse texture may be, relative to the model.
func parseFBX(data []byte, o Options) (*Mesh, [][]string, error) {
	roots, ver, err := readFBX(data)
	if err != nil {
		return nil, nil, err
	}
	if ver > 0 && ver < 7000 {
		return nil, nil, fmt.Errorf("%w: version %d.%d is too old, export as FBX 2011 or later", ErrBadFBX, ver/1000, ver%1000/100)
	}
	top := map[string]*fbxNode{}
	for _, r := range roots {
		top[r.Name] = r
	}
	if top["Objects"] == nil {
		return nil, nil, fmt.Errorf("%w: no objects", ErrBadFBX)
	}
	s := newFBXScene(top["Objects"], top["Connections"])
	b := &fbxBuild{m: &Mesh{}, s: s, parts: map[int64]int{}}
	axes := props70(top["GlobalSettings"]).canonical()
	o.UpZ = false // the file names its own axes
	for _, id := range s.order {
		n := s.objs[id]
		if n.Name != "Model" {
			continue
		}
		geos := s.children(id, "Geometry")
		if len(geos) == 0 {
			continue
		}
		mats := s.children(id, "Material")
		xf := axes.mul(s.worldOf(id, 0)).mul(props70(n).geometric())
		for _, g := range geos {
			if g.str(2) != "Mesh" {
				continue
			}
			if err := b.geometry(g, xf, mats, o); err != nil {
				return nil, nil, err
			}
		}
	}
	if len(b.m.Tris) == 0 {
		return nil, nil, fmt.Errorf("%w: no faces", ErrBadFBX)
	}
	b.m.fit()
	return b.m, b.texs, nil
}

func findTexture(fsys fs.FS, dir string, cands []string) string {
	for _, c := range cands {
		if _, err := fs.Stat(fsys, path.Join(dir, c)); err == nil {
			return c
		}
	}
	return ""
}

func loadFBX(ctx context.Context, fsys fs.FS, p string, data []byte, o Options) (*Mesh, error) {
	m, texs, err := parseFBX(data, o)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", p, err)
	}
	dir := path.Dir(p)
	for i, cands := range texs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t := findTexture(fsys, dir, cands)
		if t == "" {
			continue // the texture was not shipped with the model; the part draws in its color
		}
		m.Parts[i].Texture = t
		if avg, err := average(fsys, path.Join(dir, t)); err == nil {
			m.Parts[i].Color = color.Mul(avg, m.Parts[i].Kd)
		}
	}
	return m, nil
}

func fbxFiles(fsys fs.FS, p string) ([]string, error) {
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		return nil, fmt.Errorf("model files %s: %w", p, err)
	}
	_, texs, err := parseFBX(data, Options{})
	if err != nil {
		return nil, fmt.Errorf("model files %s: %w", p, err)
	}
	var out []string
	for _, cands := range texs {
		if t := findTexture(fsys, path.Dir(p), cands); t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, nil
}
