package dragonfall

import "embed"

//go:embed all:template
var templateFiles embed.FS // read-only build artefact, the only package level value
