package pipeline

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
)

func drawM(c color.Color) draw.Material { return draw.M(c) }
