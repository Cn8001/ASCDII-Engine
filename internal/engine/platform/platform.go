// Package platform is the contract between the engine and whatever shows its
// frames and delivers input: a terminal today, a headless writer for previews,
// the editor's viewport, and later a pixel window such as SDL.
package platform

import (
	"context"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// Frame carries both results of a render: the cells a character display
// shows, and the lit dots a pixel display can draw at full resolution.
type Frame struct {
	Cells *cell.Grid
	Dots  *raster.Buffer
	Time  float64
}

type Display interface {
	Open(ctx context.Context) error
	// Size is the display in cells; pixel backends report how many cells fit.
	Size() (cols, rows int, err error)
	Present(ctx context.Context, f Frame) error
	Close(ctx context.Context) error
}

type Input interface {
	Events(ctx context.Context) <-chan input.Event
}
