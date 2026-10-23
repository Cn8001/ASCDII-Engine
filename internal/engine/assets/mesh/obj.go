package mesh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // texture formats map_Kd may name
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

var ErrBadOBJ = errors.New("bad obj")

// clay is the color of faces without a material.
func clay() color.Color { return color.RGB(196, 190, 180) }

// ParseOBJ reads positions and faces; polygons are split into fans. It also
// returns the mtllib files the model names, relative to the model.
func ParseOBJ(r io.Reader, o Options) (*Mesh, []string, error) {
	m := &Mesh{}
	var libs []string
	part := -1
	use := func(name string) {
		for i, p := range m.Parts {
			if p.Material == name {
				part = i
				return
			}
		}
		m.Parts = append(m.Parts, Part{Material: name, Color: clay(), Kd: clay()})
		part = len(m.Parts) - 1
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		f := strings.Fields(sc.Text())
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		switch f[0] {
		case "v":
			if len(f) < 4 {
				return nil, nil, fmt.Errorf("%w: line %d: vertex needs 3 numbers", ErrBadOBJ, line)
			}
			var v [3]float64
			for i := range v {
				x, err := strconv.ParseFloat(f[i+1], 64)
				if err != nil {
					return nil, nil, fmt.Errorf("%w: line %d: %w", ErrBadOBJ, line, err)
				}
				v[i] = x
			}
			m.Pos = append(m.Pos, o.engine(geom.V3(v[0], v[1], v[2])))
		case "vt":
			if len(f) < 3 {
				return nil, nil, fmt.Errorf("%w: line %d: texture coordinate needs 2 numbers", ErrBadOBJ, line)
			}
			u, eu := strconv.ParseFloat(f[1], 64)
			v, ev := strconv.ParseFloat(f[2], 64)
			if eu != nil || ev != nil {
				return nil, nil, fmt.Errorf("%w: line %d: bad texture coordinate", ErrBadOBJ, line)
			}
			m.UV = append(m.UV, geom.V2(u, v))
		case "f":
			if len(f) < 4 {
				return nil, nil, fmt.Errorf("%w: line %d: face needs 3 corners", ErrBadOBJ, line)
			}
			idx := make([]int, 0, len(f)-1)
			uvs := make([]int, 0, len(f)-1)
			for _, c := range f[1:] {
				i, t, err := corner(c, len(m.Pos), len(m.UV))
				if err != nil {
					return nil, nil, fmt.Errorf("%w: line %d: %w", ErrBadOBJ, line, err)
				}
				idx, uvs = append(idx, i), append(uvs, t)
			}
			if part < 0 {
				use("")
			}
			for k := 2; k < len(idx); k++ {
				m.Tris = append(m.Tris, Tri{V: [3]int{idx[0], idx[k], idx[k-1]}, T: [3]int{uvs[0], uvs[k], uvs[k-1]}, Part: part})
			}
		case "usemtl":
			use(strings.Join(f[1:], " "))
		case "mtllib":
			libs = append(libs, strings.Join(f[1:], " "))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("read obj: %w", err)
	}
	if len(m.Tris) == 0 {
		return nil, nil, fmt.Errorf("%w: no faces", ErrBadOBJ)
	}
	m.fit()
	return m, libs, nil
}

// corner reads "v", "v/vt", "v//vn" or "v/vt/vn" into a position index and
// a texture coordinate index, -1 when absent; negative indices count back.
func corner(s string, n, nt int) (int, int, error) {
	v, rest, _ := strings.Cut(s, "/")
	i, err := index(v, n)
	if err != nil {
		return 0, 0, err
	}
	vt, _, _ := strings.Cut(rest, "/")
	if vt == "" {
		return i, -1, nil
	}
	t, err := index(vt, nt)
	return i, t, err
}

func index(s string, n int) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if i < 0 {
		i = n + i + 1
	}
	if i < 1 || i > n {
		return 0, fmt.Errorf("index %d out of range", i)
	}
	return i - 1, nil
}

type Material struct {
	Color   color.Color
	Texture string
}

func ParseMTL(r io.Reader) (map[string]Material, error) {
	out := map[string]Material{}
	cur := ""
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "newmtl":
			cur = strings.Join(f[1:], " ")
			out[cur] = Material{Color: clay()}
		case "Kd":
			if len(f) < 4 {
				continue
			}
			var c [3]float64
			for i := range c {
				c[i], _ = strconv.ParseFloat(f[i+1], 64)
			}
			m := out[cur]
			m.Color = color.FromFloat(c[0], c[1], c[2])
			out[cur] = m
		case "map_Kd":
			if len(f) < 2 {
				continue
			}
			m := out[cur]
			m.Texture = strings.ReplaceAll(f[len(f)-1], "\\", "/") // options like -s come first
			out[cur] = m
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read mtl: %w", err)
	}
	return out, nil
}

// Load reads a model and its materials from fsys. A textured material takes
// the texture's average color until textures are sampled per dot.
func Load(ctx context.Context, fsys fs.FS, p string, o Options) (*Mesh, error) {
	b, err := fs.ReadFile(fsys, p)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", p, err)
	}
	m, libs, err := ParseOBJ(bytes.NewReader(b), o)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", p, err)
	}
	mats := map[string]Material{}
	for _, lib := range libs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, err := fs.ReadFile(fsys, path.Join(path.Dir(p), lib))
		if err != nil {
			continue // a model without its .mtl still draws, in clay
		}
		ms, err := ParseMTL(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", lib, err)
		}
		for k, v := range ms {
			mats[k] = v
		}
	}
	for i, part := range m.Parts {
		mat, ok := mats[part.Material]
		if !ok {
			continue
		}
		m.Parts[i].Color, m.Parts[i].Kd, m.Parts[i].Texture = mat.Color, mat.Color, mat.Texture
		if mat.Texture == "" {
			continue
		}
		if avg, err := average(fsys, path.Join(path.Dir(p), mat.Texture)); err == nil {
			m.Parts[i].Color = color.Mul(avg, mat.Color)
		}
	}
	return m, nil
}

// Files lists what a model needs next to it: its .mtl files and their
// textures, relative to the model's folder.
func Files(fsys fs.FS, p string) ([]string, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, fmt.Errorf("model files %s: %w", p, err)
	}
	defer f.Close()
	_, libs, err := ParseOBJ(f, Options{})
	if err != nil {
		return nil, fmt.Errorf("model files %s: %w", p, err)
	}
	out := libs
	for _, lib := range libs {
		raw, err := fs.ReadFile(fsys, path.Join(path.Dir(p), lib))
		if err != nil {
			continue
		}
		ms, err := ParseMTL(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		for _, m := range ms {
			if m.Texture != "" {
				out = append(out, m.Texture)
			}
		}
	}
	return out, nil
}

func average(fsys fs.FS, p string) (color.Color, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return color.Color{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return color.Color{}, fmt.Errorf("decode %s: %w", p, err)
	}
	b := img.Bounds()
	step := max(1, max(b.Dx(), b.Dy())/64)
	var r, g, bl, n float64
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			if ca < 0x8000 {
				continue
			}
			r, g, bl, n = r+float64(cr), g+float64(cg), bl+float64(cb), n+1
		}
	}
	if n == 0 {
		return color.Color{}, fmt.Errorf("decode %s: fully transparent", p)
	}
	k := n * 0xffff
	return color.FromFloat(r/k, g/k, bl/k), nil
}
