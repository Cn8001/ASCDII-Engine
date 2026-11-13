package vm

import (
	lua "github.com/yuin/gopher-lua"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/input"
	"github.com/Cn8001/ASCDII-Engine/internal/engine/render/raster"
)

// mouse() returns nil until the display has reported a pointer, else a
// table: x, y (the cell), left, middle, right (held), clicked_left,
// clicked_middle, clicked_right (went down this frame), wheel (steps, up
// positive) and world, the ground point (z = 0) under the pointer as seen
// by the main camera.
func (v *VM) mouseGlobal(L *lua.LState) int {
	if v.ctx.Input == nil {
		L.Push(lua.LNil)
		return 1
	}
	m := v.ctx.Input.Mouse()
	if !m.Seen {
		L.Push(lua.LNil)
		return 1
	}
	t := L.NewTable()
	t.RawSetString("x", lua.LNumber(m.X))
	t.RawSetString("y", lua.LNumber(m.Y))
	names := map[input.Button]string{input.ButtonLeft: "left", input.ButtonMiddle: "middle", input.ButtonRight: "right"}
	for b, name := range names {
		t.RawSetString(name, lua.LBool(m.Held[b]))
		t.RawSetString("clicked_"+name, lua.LBool(m.Clicked[b]))
	}
	t.RawSetString("wheel", lua.LNumber(m.Wheel))
	cols, rows := v.ctx.Input.Screen()
	if cam, ok := v.ctx.Scene.View(); ok && cols > 0 && rows > 0 {
		view := cam.View(cols*raster.SubW, rows*raster.SubH)
		p := view.Unproject(float64(m.X*raster.SubW)+raster.SubW/2.0, float64(m.Y*raster.SubH)+raster.SubH/2.0, 0)
		t.RawSetString("world", v.vec3(p.X, p.Y, 0))
	}
	L.Push(t)
	return 1
}
