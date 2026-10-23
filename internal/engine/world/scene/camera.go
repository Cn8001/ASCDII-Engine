package scene

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
)

// Viewer is a camera component. The first enabled one on a visible object
// decides what the game shows; scenes without one use Scene.Camera.
type Viewer interface {
	View(t Transform) view.Camera
	Place(o *GameObject, c view.Camera)
}

// MainCamera is the object holding the camera that is in use, or nil.
func (s *Scene) MainCamera() (*GameObject, Viewer) {
	var obj *GameObject
	var cam Viewer
	s.Walk(func(e *GameObject, visible bool) bool {
		if obj != nil || !visible {
			return false
		}
		for _, c := range e.Components {
			if v, ok := c.(Viewer); ok && e.Enabled(c) {
				obj, cam = e, v
				return false
			}
		}
		return true
	})
	return obj, cam
}

// View is what the game shows.
func (s *Scene) View() view.Camera {
	if o, c := s.MainCamera(); o != nil {
		return c.View(o.World())
	}
	return s.Camera
}

// SetView points the game camera: it moves the camera object, or sets the
// scene camera when there is none.
func (s *Scene) SetView(c view.Camera) {
	if o, v := s.MainCamera(); o != nil {
		v.Place(o, c)
		return
	}
	s.Camera = c
}

// FollowView moves the game camera toward target at rate. A camera object
// that is a child of follower already moves with it, so it is left alone.
func (s *Scene) FollowView(follower *GameObject, target geom.Vec3, dt, rate float64) {
	if o, _ := s.MainCamera(); o != nil {
		for p := o.Parent(); p != nil; p = p.Parent() {
			if p == follower {
				return
			}
		}
	}
	c := s.View()
	c.Follow(target, dt, rate)
	s.SetView(c)
}

// CameraProps edits whichever camera is in use through the scene camera's
// property names, for scripts and tools that address "the camera".
func (s *Scene) CameraProps() prop.Editable { return cameraProps{s} }

type cameraProps struct{ s *Scene }

func (c cameraProps) Props() []prop.Prop {
	v := c.s.View()
	return v.Props()
}

func (c cameraProps) Set(name string, value any) error {
	v := c.s.View()
	if err := v.Set(name, value); err != nil {
		return err
	}
	c.s.SetView(v)
	return nil
}
