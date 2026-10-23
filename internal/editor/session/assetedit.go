package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	stdcolor "image/color"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/texture"
)

var ErrNoSettings = errors.New("asset has no settings")

// CreateMaterial writes a white material and returns its project path.
func (s *Session) CreateMaterial(ctx context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel := strings.TrimSuffix(filepath.ToSlash(name), material.Ext)
	if locked(rel) {
		return "", lockedErr(rel)
	}
	rel = s.freeName(rel + material.Ext)
	m := material.Default()
	var b bytes.Buffer
	if err := material.Encode(&b, m); err != nil {
		return "", err
	}
	if err := s.writeAsset(ctx, rel, b.Bytes()); err != nil {
		return "", err
	}
	s.lib.PutMaterial(rel, m)
	return rel, nil
}

// settings is the editable side of an asset: a material's properties, or
// the import settings of a texture or model, which live in its .meta.
type settings struct {
	kind  asset.Kind
	props []prop.Prop
	set   func(name string, v any) error
	save  func(ctx context.Context) error
}

func (s *Session) settingsOf(ref string) (settings, error) {
	p := s.lib.Resolve(ref)
	k, _ := asset.Classify(p)
	fsys := os.DirFS(s.dir)
	switch k {
	case asset.Material:
		m, err := s.lib.Material(p)
		if err != nil {
			return settings{}, err
		}
		return settings{kind: k, props: m.Props(), set: m.Set, save: func(ctx context.Context) error {
			var b bytes.Buffer
			if err := material.Encode(&b, m); err != nil {
				return err
			}
			return writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(p)), b.Bytes())
		}}, nil
	case asset.Texture:
		o := asset.TextureOptions(fsys, p)
		if o.MaxSize == 0 {
			o.MaxSize = 256
		}
		if o.Filter == "" {
			o.Filter = texture.Linear
		}
		if o.Cutoff == 0 {
			o.Cutoff = .5
		}
		filter := string(o.Filter)
		b := prop.Bind().Int("maxSize", &o.MaxSize, 8, 2048).Enum("filter", &filter, string(texture.Linear), string(texture.Nearest)).
			Float("alphaCutoff", &o.Cutoff, .01, 1)
		return settings{kind: k, props: b.Props(), set: b.Set, save: func(ctx context.Context) error {
			o.Filter = texture.Filter(filter)
			if err := asset.WriteImport(ctx, s.dir, p, o); err != nil {
				return err
			}
			t, err := asset.LoadTexture(os.DirFS(s.dir), p)
			if err != nil {
				return err
			}
			s.lib.PutTexture(p, t)
			return nil
		}}, nil
	case asset.Model:
		o := asset.ModelOptions(fsys, p)
		if o.Scale == 0 {
			o.Scale = 1
		}
		b := prop.Bind().Bool("upZ", &o.UpZ).Float("scale", &o.Scale, .001, 1000)
		return settings{kind: k, props: b.Props(), set: b.Set, save: func(ctx context.Context) error {
			if err := asset.WriteImport(ctx, s.dir, p, o); err != nil {
				return err
			}
			m, err := mesh.Load(ctx, os.DirFS(s.dir), p, o)
			if err != nil {
				return err
			}
			s.lib.PutModel(p, m)
			return nil
		}}, nil
	}
	return settings{}, fmt.Errorf("%s: %w", p, ErrNoSettings)
}

// AssetProps lists what the inspector edits for an asset in the Project window.
func (s *Session) AssetProps(ref string) ([]prop.Prop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return nil, err
	}
	st, err := s.settingsOf(ref)
	if err != nil {
		return nil, err
	}
	return st.props, nil
}

// SetAssetProp changes and saves one setting; textures and models are
// imported again so the scene shows the change at once.
func (s *Session) SetAssetProp(ctx context.Context, ref, name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if p := s.lib.Resolve(ref); locked(p) {
		return lockedErr(p)
	}
	st, err := s.settingsOf(ref)
	if err != nil {
		return err
	}
	for _, p := range st.props {
		if str, isStr := v.(string); isStr && p.Name == name && p.Type.Ref() {
			v = s.lib.Ref(str)
		}
	}
	if err := st.set(name, v); err != nil {
		return err
	}
	if err := st.save(ctx); err != nil {
		return err
	}
	s.touch()
	return nil
}

// Thumb is a small PNG of a texture as a data URL, for the Project window.
func (s *Session) Thumb(ref string, size int) (string, error) {
	s.mu.Lock()
	p := filepath.Join(s.dir, filepath.FromSlash(s.lib.Resolve(ref)))
	s.mu.Unlock()
	f, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("thumb: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("thumb: %w", err)
	}
	b := img.Bounds()
	size = max(8, min(size, 256))
	k := float64(size) / float64(max(b.Dx(), b.Dy()))
	w, h := max(1, int(float64(b.Dx())*k)), max(1, int(float64(b.Dy())*k))
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := stdcolor.NRGBAModel.Convert(img.At(b.Min.X+x*b.Dx()/w, b.Min.Y+y*b.Dy()/h))
			out.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return "", fmt.Errorf("thumb: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ConvertToSprite writes a texture as a sprite cols cells wide next to it.
func (s *Session) ConvertToSprite(ctx context.Context, ref string, cols int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	src := s.lib.Resolve(ref)
	f, err := os.Open(filepath.Join(s.dir, filepath.FromSlash(src)))
	if err != nil {
		return "", fmt.Errorf("convert %s: %w", src, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("convert %s: %w", src, err)
	}
	art, err := asset.SpriteFromImage(img, cols)
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(path.Base(src), path.Ext(src))
	var b bytes.Buffer
	if err := asset.EncodeJSON(&b, name, art); err != nil {
		return "", err
	}
	rel := s.freeName(path.Join(path.Dir(src), name+asset.SpriteExt))
	if err := s.writeAsset(ctx, rel, b.Bytes()); err != nil {
		return "", err
	}
	s.lib.PutSprite(rel, art)
	return rel, nil
}

// ApplyMaterial puts a material on the first component of id that takes one.
func (s *Session) ApplyMaterial(id uint64, ref string) error {
	s.mu.Lock()
	o, err := s.current().ByID(id)
	name := ""
	if err == nil {
		for _, p := range o.Props() {
			if p.Type == prop.MatRef && !p.Locked {
				name = p.Name
				break
			}
		}
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("%s takes no material: %w", o.Name, ErrNotReference)
	}
	return s.SetProp(id, name, ref)
}

// MaterialAt applies a material to whatever is drawn at a viewport cell, the
// drag and drop way of texturing an object.
func (s *Session) MaterialAt(ref string, col, row int) (uint64, error) {
	s.mu.Lock()
	id := uint64(s.frame.Buf.Pick(col, row))
	s.mu.Unlock()
	if id == 0 {
		return 0, fmt.Errorf("nothing under the pointer: %w", ErrNotReference)
	}
	return id, s.ApplyMaterial(id, ref)
}

// CreateController writes a one state animator controller.
func (s *Session) CreateController(ctx context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel := strings.TrimSuffix(filepath.ToSlash(name), animator.Ext)
	if locked(rel) {
		return "", lockedErr(rel)
	}
	rel = s.freeName(rel + animator.Ext)
	c := animator.Default()
	var b bytes.Buffer
	if err := animator.Encode(&b, c); err != nil {
		return "", err
	}
	if err := s.writeAsset(ctx, rel, b.Bytes()); err != nil {
		return "", err
	}
	s.lib.PutController(rel, c)
	return rel, nil
}

func (s *Session) Controller(ref string) (*animator.Controller, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lib.Controller(ref)
}

// SetController saves a whole controller, as the animator window edits it.
func (s *Session) SetController(ctx context.Context, ref string, c *animator.Controller) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	p := s.lib.Resolve(ref)
	if locked(p) {
		return lockedErr(p)
	}
	if _, err := s.lib.Controller(p); err != nil {
		return err
	}
	var b bytes.Buffer
	if err := animator.Encode(&b, c); err != nil {
		return err
	}
	if err := writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(p)), b.Bytes()); err != nil {
		return err
	}
	s.lib.PutController(p, c)
	s.touch()
	return nil
}

// AnimatorState is what an object's animator is doing, for the animator
// window to light up the active state while the game runs.
func (s *Session) AnimatorState(id uint64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.current().ByID(id)
	if err != nil {
		return ""
	}
	if p, ok := prop.Get(o, "animator.state"); ok {
		v, _ := p.Value.(string)
		return v
	}
	return ""
}
