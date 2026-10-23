package scene

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

// References name objects by FID, which is saved with the file, unlike ID:
//
//	"12"     the scene's own object 12
//	"12/5"   object 5 of the prefab instanced as object 12
//	"~5"     object 5 of the prefab this reference itself came from
//
// A component reference appends its key: "12#light", "~3#door".

// scope is the prefab instance whose file o's components came from.
func scope(o *GameObject) *GameObject {
	if o == nil {
		return nil
	}
	if o.inherited {
		return o.InstanceRoot()
	}
	if o.Prefab != "" {
		return o
	}
	return nil
}

// fileFID is o's id inside the file that defines it.
func fileFID(o *GameObject) uint64 {
	if o.Prefab != "" && !o.inherited {
		return o.srcFID
	}
	return o.FID
}

// RefOf is how a component on from stores a reference to to.
func RefOf(from, to *GameObject) string {
	if to == nil {
		return ""
	}
	r := scope(to)
	if r != nil && r == scope(from) {
		return "~" + strconv.FormatUint(fileFID(to), 10)
	}
	if to.inherited && r != nil {
		return fmt.Sprintf("%d/%d", r.FID, to.FID)
	}
	return strconv.FormatUint(to.FID, 10)
}

// CompRefOf is RefOf for a component on to.
func CompRefOf(from, to *GameObject, c Component) string {
	return RefOf(from, to) + "#" + to.Key(c)
}

func (s *Scene) own(fid uint64) *GameObject {
	var out *GameObject
	s.Walk(func(o *GameObject, _ bool) bool {
		if o.inherited {
			return false
		}
		if o.FID == fid && o != s.Root {
			out = o
		}
		return out == nil
	})
	return out
}

func inside(root *GameObject, fid uint64) *GameObject {
	for _, c := range root.Children {
		if c.inherited && c.FID == fid {
			return c
		}
		if found := inside(c, fid); found != nil {
			return found
		}
	}
	return nil
}

// Lookup resolves a reference held by a component on from. Values that are
// not references are taken as object names, as files before references had.
func (s *Scene) Lookup(from *GameObject, ref string) *GameObject {
	ref, _, _ = strings.Cut(ref, "#")
	if ref == "" {
		return nil
	}
	if rest, ok := strings.CutPrefix(ref, "~"); ok {
		r := scope(from)
		fid, err := strconv.ParseUint(rest, 10, 64)
		if r == nil || err != nil {
			return nil
		}
		if fid == r.srcFID {
			return r
		}
		return inside(r, fid)
	}
	a, b, nested := strings.Cut(ref, "/")
	fa, err := strconv.ParseUint(a, 10, 64)
	if err != nil {
		return s.Find(ref)
	}
	o := s.own(fa)
	if !nested || o == nil {
		return o
	}
	fb, err := strconv.ParseUint(b, 10, 64)
	if err != nil {
		return nil
	}
	return inside(o, fb)
}

// LookupComponent resolves "ref#key"; a bare object reference picks its
// first component of kind, or its main component when kind is "".
func (s *Scene) LookupComponent(from *GameObject, ref, kind string) (*GameObject, Component) {
	o := s.Lookup(from, ref)
	if o == nil {
		return nil, nil
	}
	if _, key, ok := strings.Cut(ref, "#"); ok {
		return o, o.ComponentByKey(key)
	}
	if kind == "" {
		return o, o.Main()
	}
	return o, o.Component(kind)
}

// display names what a reference property points at.
func (o *GameObject) display(p prop.Prop) string {
	ref, _ := p.Value.(string)
	if ref == "" || o.scene == nil {
		return ""
	}
	if p.Type == prop.ObjectRef {
		if t := o.scene.Lookup(o, ref); t != nil {
			return t.Name
		}
		return "Missing"
	}
	kind := ""
	if len(p.Options) > 0 {
		kind = p.Options[0]
	}
	t, c := o.scene.LookupComponent(o, ref, kind)
	if t == nil || c == nil {
		return "Missing"
	}
	return t.Name + " (" + c.Kind() + ")"
}

// number gives every own object below o (o included) an FID that is not
// taken yet, keeping the ones it already has where possible.
func (s *Scene) number(o *GameObject, taken map[uint64]bool) {
	if !o.inherited && o != s.Root {
		if o.FID == 0 || taken[o.FID] {
			o.FID = s.newFID(taken)
		}
		taken[o.FID] = true
	}
	for _, c := range o.Children {
		s.number(c, taken)
	}
}

func (s *Scene) newFID(taken map[uint64]bool) uint64 {
	for {
		if f := s.newFIDs(); f != 0 && !taken[f] {
			return f
		}
	}
}

// FIDSource picks ids for new objects.
type FIDSource func() uint64

// RandomFIDs are what scenes use: two branches that both add objects then
// merge without clashing ids.
// They stay below 2^53 so JavaScript reads them exactly.
func RandomFIDs() FIDSource {
	return func() uint64 {
		var b [8]byte
		_, _ = rand.Read(b[:]) // crypto/rand does not fail on supported systems
		return binary.LittleEndian.Uint64(b[:]) & (1<<53 - 1)
	}
}

// CountingFIDs number objects 1, 2, 3, for generated files that must come
// out the same on every run.
func CountingFIDs() FIDSource {
	n := uint64(0)
	return func() uint64 {
		n++
		return n
	}
}

// fids lists the FIDs of own objects, except those below skip.
func (s *Scene) fids(skip *GameObject) map[uint64]bool {
	out := map[uint64]bool{}
	s.Walk(func(o *GameObject, _ bool) bool {
		if o == skip || o.inherited {
			return false
		}
		if o != s.Root {
			out[o.FID] = true
		}
		return true
	})
	return out
}

// RewriteRefs maps every object and component reference held below root
// through fn, for moves that change how a reference must be written.
func RewriteRefs(root *GameObject, fn func(owner *GameObject, ref string) string) {
	var walk func(o *GameObject)
	walk = func(o *GameObject) {
		for _, c := range o.Components {
			for _, p := range c.Props() {
				if p.Type != prop.ObjectRef && p.Type != prop.CompRef {
					continue
				}
				v, _ := p.Value.(string)
				if v == "" {
					continue
				}
				if n := fn(o, v); n != v {
					_ = c.Set(p.Name, n)
				}
			}
		}
		for _, c := range o.Children {
			walk(c)
		}
	}
	walk(root)
}

// splitRef separates "ref#key" into the object part and the "#key" suffix.
func splitRef(ref string) (string, string) {
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		return ref[:i], ref[i:]
	}
	return ref, ""
}

// Localize rewrites references inside root that point into root to the
// prefab local form, before root is saved as a prefab.
func Localize(root *GameObject) {
	in := map[uint64]bool{}
	var mark func(o *GameObject)
	mark = func(o *GameObject) {
		if !o.inherited {
			in[o.FID] = true
		}
		for _, c := range o.Children {
			mark(c)
		}
	}
	mark(root)
	RewriteRefs(root, func(_ *GameObject, ref string) string {
		obj, key := splitRef(ref)
		fid, err := strconv.ParseUint(obj, 10, 64)
		if err != nil || !in[fid] {
			return ref
		}
		return "~" + obj + key
	})
}

// relink fixes references after the objects in old, keyed to their previous
// file FIDs, became own objects of the scene when root was unpacked.
func (s *Scene) relink(root *GameObject, old map[*GameObject]uint64) {
	byOld := map[uint64]*GameObject{}
	for o, f := range old {
		byOld[f] = o
	}
	inst := strconv.FormatUint(root.FID, 10)
	RewriteRefs(s.Root, func(owner *GameObject, ref string) string {
		obj, key := splitRef(ref)
		local, isLocal := strings.CutPrefix(obj, "~")
		a, b, nested := strings.Cut(obj, "/")
		switch {
		case isLocal && within(owner, root):
		case nested && a == inst:
			local = b
		default:
			return ref
		}
		f, err := strconv.ParseUint(local, 10, 64)
		if t := byOld[f]; err == nil && t != nil {
			return strconv.FormatUint(t.FID, 10) + key
		}
		return ref
	})
}

func within(o, root *GameObject) bool {
	for a := o; a != nil; a = a.parent {
		if a == root {
			return true
		}
	}
	return false
}

// FIDs records the FIDs of the own objects below root before a Graft may
// renumber them; Remap then points references among them at the new ones.
func FIDs(root *GameObject) map[*GameObject]uint64 {
	out := map[*GameObject]uint64{}
	var walk func(o *GameObject)
	walk = func(o *GameObject) {
		if !o.inherited {
			out[o] = o.FID
		}
		for _, c := range o.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// Remap makes references inside a copied subtree point at the copies, not
// at the originals they were copied from.
func Remap(root *GameObject, before map[*GameObject]uint64) {
	moved := map[string]string{}
	for o, f := range before {
		moved[strconv.FormatUint(f, 10)] = strconv.FormatUint(o.FID, 10)
	}
	RewriteRefs(root, func(_ *GameObject, ref string) string {
		obj, key := splitRef(ref)
		a, rest, nested := strings.Cut(obj, "/")
		n, ok := moved[a]
		if !ok {
			return ref
		}
		if nested {
			return n + "/" + rest + key
		}
		return n + key
	})
}
