// Package frontend embeds the built editor UI for the Wails asset server.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS // read-only build artefact produced by vite

func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // the embed pattern guarantees the directory exists
	}
	return sub
}
