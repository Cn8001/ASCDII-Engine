package asset

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif" // texture formats
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"path"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/texture"
)

// TextureOptions reads a texture's import settings from its .meta.
func TextureOptions(fsys fs.FS, p string) texture.Options {
	var o texture.Options
	if m, err := ReadMeta(fsys, p); err == nil && len(m.Import) > 0 {
		_ = json.Unmarshal(m.Import, &o) // bad settings fall back to defaults
	}
	return o
}

// ModelOptions reads a model's import settings from its .meta.
func ModelOptions(fsys fs.FS, p string) mesh.Options {
	var o mesh.Options
	if m, err := ReadMeta(fsys, p); err == nil && len(m.Import) > 0 {
		_ = json.Unmarshal(m.Import, &o)
	}
	return o
}

func LoadTexture(fsys fs.FS, p string) (*texture.Texture, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, fmt.Errorf("load texture %s: %w", p, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("load texture %s: %w", p, err)
	}
	return texture.FromImage(img, TextureOptions(fsys, p)), nil
}

func (l *Library) PutTexture(name string, t *texture.Texture) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tex[path.Clean(name)] = t
}

func (l *Library) Texture(ref string) (*texture.Texture, error) {
	p := l.Resolve(ref)
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.tex[p]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return t, nil
}

func (l *Library) PutMaterial(name string, m *material.Material) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mats[path.Clean(name)] = m
}

// Material returns the shared asset; edit it only through the editor.
func (l *Library) Material(ref string) (*material.Material, error) {
	p := l.Resolve(ref)
	l.mu.RLock()
	defer l.mu.RUnlock()
	m, ok := l.mats[p]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return m, nil
}

// DrawMaterial resolves a material asset, texture included, for drawing.
func (l *Library) DrawMaterial(ref string) (draw.Material, bool) {
	m, err := l.Material(ref)
	if err != nil {
		return draw.Material{}, false
	}
	out := draw.Material{Albedo: m.Albedo, Emissive: m.Emissive, Unlit: m.Unlit, Tiling: m.Tiling, Offset: m.Offset}
	if m.Texture != "" {
		if t, err := l.Texture(m.Texture); err == nil {
			out.Texture = t
		}
	}
	return out, true
}
