package scene

import (
	"errors"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
)

var ErrNoCamera = errors.New("the scene has no camera")

// Viewer is a camera component. The first enabled one on a visible object
// decides what the game shows; a scene without one shows nothing.
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

// View is what the game shows; ok is false without a camera.
func (s *Scene) View() (c view.Camera, ok bool) {
	if o, v := s.MainCamera(); o != nil {
		return v.View(o.World()), true
	}
	return view.Camera{}, false
}

// SetView points the game camera by moving the camera object.
func (s *Scene) SetView(c view.Camera) error {
	o, v := s.MainCamera()
	if o == nil {
		return ErrNoCamera
	}
	v.Place(o, c)
	return nil
}

// FollowView moves the game camera toward target at rate. A camera object
// that is a child of follower already moves with it, so it is left alone.
func (s *Scene) FollowView(follower *GameObject, target geom.Vec3, dt, rate float64) {
	o, v := s.MainCamera()
	if o == nil {
		return
	}
	for p := o.Parent(); p != nil; p = p.Parent() {
		if p == follower {
			return
		}
	}
	c := v.View(o.World())
	c.Follow(target, dt, rate)
	v.Place(o, c)
}

// CameraProps edits the camera in use through the view's property names, for
// scripts and tools that address "the camera"; nil without one.
func (s *Scene) CameraProps() prop.Editable {
	if o, _ := s.MainCamera(); o == nil {
		return nil
	}
	return cameraProps{s}
}

type cameraProps struct{ s *Scene }

func (c cameraProps) Props() []prop.Prop {
	v, _ := c.s.View()
	return v.Props()
}

func (c cameraProps) Set(name string, value any) error {
	v, ok := c.s.View()
	if !ok {
		return ErrNoCamera
	}
	if err := v.Set(name, value); err != nil {
		return err
	}
	return c.s.SetView(v)
}
