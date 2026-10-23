// Package winbrand gives a Windows build the game's icon, name and version
// in place of the player's. Only Windows programs link the resource writer.
package winbrand

import (
	"fmt"
	"image"
	_ "image/gif" // every image format the project accepts as a texture
	_ "image/jpeg"
	_ "image/png"
	"io/fs"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

// Branding is what a Windows build shows in Explorer, the taskbar and the
// file properties.
type Branding struct {
	Name, Version string
	Icon          image.Image // nil keeps the player's icon
}

// For reads the branding of a project from its manifest.
func For(m project.Manifest, fsys fs.FS, lib *asset.Library) (Branding, error) {
	b := Branding{Name: m.Name, Version: m.Version}
	if m.Icon == "" {
		return b, nil
	}
	img, err := LoadIcon(fsys, lib.Resolve(m.Icon))
	if err != nil {
		return b, err
	}
	b.Icon = img
	return b, nil
}

// LoadIcon decodes the PNG, JPEG or GIF at name in fsys.
func LoadIcon(fsys fs.FS, name string) (image.Image, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("icon %s: %w", name, err)
	}
	return img, nil
}
