package dragonfall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// TemplateFiles renders every generated file of the template project.
func TemplateFiles() (map[string][]byte, error) {
	reg, _ := project.Registry([]project.Module{Module()})
	s, err := BuildScene(reg)
	if err != nil {
		return nil, err
	}
	var sc, pf bytes.Buffer
	if err := scene.Encode(&sc, s); err != nil {
		return nil, err
	}
	if err := scene.EncodePrefab(&pf, Lantern()); err != nil {
		return nil, err
	}
	m := project.Manifest{Format: 1, Name: "Dragonfall", Version: "0.1.0", Engine: version.Engine, Start: StartScene, Cols: 160, Rows: 50, FPS: 30, Modules: []string{Name}}
	mb, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	files := map[string][]byte{
		"Dragonfall" + project.Ext: append(mb, '\n'),
		StartScene:                 sc.Bytes(),
		LanternPrefab:              pf.Bytes(),
		CrystalScript:              []byte(crystalLua),
		DragonScript:               []byte(dragonLua),
		".gitignore":               []byte(project.BuildDir + "/\n"),
		"README.md":                []byte("# Dragonfall\n\nThe ASCDII Engine sample: a 2.5D Braille valley with a keep, a sleeping dragon and a knight.\n\n```bash\nascdii run .\n```\n"),
	}
	for _, sp := range []string{"sprites/pine.sprite.json"} {
		files[sp+asset.MetaExt] = asset.MetaBytes(TemplateGUID(sp))
	}
	for name := range files {
		if _, ok := asset.Classify(name); ok && !strings.HasSuffix(name, asset.MetaExt) && name != "Dragonfall"+project.Ext {
			files[name+asset.MetaExt] = asset.MetaBytes(TemplateGUID(name))
		}
	}
	return files, nil
}
