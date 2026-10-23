package scene

import (
	"bytes"
	"errors"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"math"
	"strings"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
)

type crate struct {
	Size geom.Vec3
	Tint color.Color
}

func newCrate() Component { return &crate{Size: geom.V3(2, 2, 2), Tint: color.RGB(160, 110, 60)} }

// eye is a minimal camera component; the real one lives in components.
type eye struct{ c view.Camera }

func (*eye) Kind() string          { return "eye" }
func (*eye) Props() []prop.Prop    { return nil }
func (*eye) Set(string, any) error { return nil }
func (e *eye) View(t Transform) view.Camera {
	c := e.c
	c.Target = t.Pos
	return c
}
func (e *eye) Place(o *GameObject, c view.Camera) { e.c, o.Local.Pos = c, c.Target }

func (c *crate) Kind() string { return "crate" }
func (c *crate) bind() *prop.Binder {
	return prop.Bind().Vec3("size", &c.Size).Color("tint", &c.Tint)
}
func (c *crate) Props() []prop.Prop                { return c.bind().Props() }
func (c *crate) Set(n string, v any) error         { return c.bind().Set(n, v) }
func (c *crate) Bounds(t Transform) geom.AABB      { return geom.Footprint(t.Pos, c.Size.Mul(t.scale())) }
func (c *crate) Draw(d *draw.Context, t Transform) { d.Box(c.Bounds(t), draw.M(c.Tint)) }

type counter struct {
	Keys  int
	Ticks int
}

func (c *counter) Kind() string                                { return "counter" }
func (c *counter) Props() []prop.Prop                          { return prop.Bind().Int("keys", &c.Keys, 0, 0).Props() }
func (c *counter) Set(n string, v any) error                   { return prop.Bind().Int("keys", &c.Keys, 0, 0).Set(n, v) }
func (c *counter) Update(*Ctx, *GameObject, float64)           { c.Ticks++ }
func (c *counter) Key(_ *Ctx, _ *GameObject, _ input.Key) bool { c.Keys++; return true }

func registry() *Registry {
	r := NewRegistry()
	r.Register("crate", newCrate, Meta{})
	r.Register("counter", func() Component { return &counter{} }, Meta{})
	r.Register("link", func() Component { return &link{} }, Meta{})
	return r
}

type prefabMap map[string][]byte

func (p prefabMap) Prefab(name string) (*GameObject, error) {
	return DecodePrefab(bytes.NewReader(p[name]), registry())
}

func TestUnit_ChildTransformFollowsParent(t *testing.T) {
	// Arrange
	s := New("t")
	parent := s.Add("p", nil).At(10, 0, 0).Scaled(2)
	parent.Local.Flip = true

	// Act
	child := parent.Add("c", newCrate()).At(1, 2, 0)

	// Assert
	w := child.World()
	if w.Pos != geom.V3(8, 4, 0) || w.Scale != geom.V3(2, 2, 2) || !w.Flip {
		t.Fatalf("got %+v", w)
	}
}

func TestUnit_BlockedUsesDrawBounds(t *testing.T) {
	// Arrange
	s := New("t")
	box := s.Add("box", newCrate()).At(5, 5, 0).Blocking()

	// Act
	inside := s.Blocked(geom.V2(5.5, 5.5), nil)
	outside := s.Blocked(geom.V2(7.5, 5), nil)
	self := s.Blocked(geom.V2(5.5, 5.5), box)

	// Assert
	if !inside || outside || self {
		t.Fatalf("inside %v outside %v self %v", inside, outside, self)
	}
}

func TestUnit_ReparentRefusesCycles(t *testing.T) {
	// Arrange
	s := New("t")
	a := s.Add("a", nil)
	b := a.Add("b", nil)

	// Act
	err := s.Reparent(a.ID, b.ID, 0, false)

	// Assert
	if !errors.Is(err, ErrInvalidParent) {
		t.Fatalf("got %v", err)
	}
	if _, err := s.ByID(999); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_ReparentKeepsWorldTransform(t *testing.T) {
	// Arrange
	s := New("t")
	p := s.Add("p", nil).At(10, 4, 0)
	p.Local.Rot, p.Local.Scale = geom.V3(0, 0, 90), geom.V3(2, 2, 2)
	c := s.Add("c", nil).At(3, 1, 2)
	c.Local.Rot = geom.V3(0, 0, 30)
	before := c.World()

	// Act
	err := s.Reparent(c.ID, p.ID, -1, true)
	after := c.World()

	// Assert
	if err != nil || after.Pos.Sub(before.Pos).Len() > 1e-9 || after.Rot.Sub(before.Rot).Len() > 1e-9 || after.Scale.Sub(before.Scale).Len() > 1e-9 {
		t.Fatalf("err %v before %+v after %+v local %+v", err, before, after, c.Local)
	}
}

func TestUnit_SceneJSONRoundTrip(t *testing.T) {
	// Arrange
	s := New("demo")
	box := s.Add("box", newCrate()).At(1, 2, 3).With(&counter{Keys: 4})
	box.Add("lid", newCrate()).At(0, 0, 2)
	_ = box.Set("tint", "#112233")
	s.Animate("box", anim.Tween("pos.z", 0.0, 1.0, 1, anim.Linear))
	var buf bytes.Buffer

	// Act
	err := Encode(&buf, s)
	back, errBack := Decode(&buf, registry(), nil)

	// Assert
	if err != nil || errBack != nil {
		t.Fatal(err, errBack)
	}
	got := back.Find("box")
	if got == nil || got.Local.Pos != geom.V3(1, 2, 3) || got.Component("crate").(*crate).Tint != color.RGB(17, 34, 51) {
		t.Fatalf("box %+v", got)
	}
	if got.Component("counter").(*counter).Keys != 4 || len(got.Children) != 1 || len(back.Clips) != 1 {
		t.Fatalf("lost data %+v", got)
	}
}

func TestUnit_DecodeRejectsUnknownKind(t *testing.T) {
	// Arrange
	doc := `{"version":1,"name":"x","entities":[{"name":"a","node":{"kind":"ghost"}}]}`

	// Act
	_, err := Decode(bytes.NewBufferString(doc), registry(), nil)

	// Assert
	if !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_PrefabInstancesAreRebuiltAndNotSaved(t *testing.T) {
	// Arrange
	src := New("src")
	tower := src.Add("tower", newCrate())
	tower.Add("flag", newCrate()).At(0, 0, 3)
	var pf bytes.Buffer
	if err := EncodePrefab(&pf, tower); err != nil {
		t.Fatal(err)
	}
	s := New("level")
	s.UsePrefabs(prefabMap{"tower.prefab.json": pf.Bytes()})
	inst := s.Add("north tower", nil).At(20, 0, 0)
	inst.Prefab = "tower.prefab.json"

	// Act
	err := s.Instantiate()
	var saved bytes.Buffer
	errSave := Encode(&saved, s)

	// Assert
	if err != nil || errSave != nil || inst.Component("crate") == nil || len(inst.Children) != 1 || !inst.Children[0].Inherited() {
		t.Fatalf("instance %+v err %v", inst, err)
	}
	if w := inst.Children[0].World().Pos; w != geom.V3(20, 0, 3) {
		t.Fatalf("child world %+v", w)
	}
	if bytes.Contains(saved.Bytes(), []byte("flag")) {
		t.Fatal("inherited child was saved")
	}
	if err := s.Remove(inst.Children[0].ID); !errors.Is(err, ErrPrefabLocked) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_UpdateRoutesKeysAndAppliesClips(t *testing.T) {
	// Arrange
	s := New("t")
	e := s.Add("box", newCrate()).With(&counter{})
	s.Animate("box", anim.Tween("pos.z", 0.0, 2.0, 1, anim.Linear))

	// Act
	err := s.Update(&Ctx{Scene: s}, .5, []input.Key{'a', 'b'})

	// Assert
	c := e.Component("counter").(*counter)
	if err != nil || c.Keys != 2 || c.Ticks != 1 || math.Abs(e.Local.Pos.Z-1) > 1e-9 {
		t.Fatalf("err %v counter %+v pos %+v", err, c, e.Local.Pos)
	}
}

func TestUnit_PropsGroupBehaviourWithPrefix(t *testing.T) {
	// Arrange
	s := New("t")
	e := s.Add("box", newCrate()).With(&counter{})

	// Act
	err := e.Set("counter.keys", 7)
	ps := e.Props()

	// Assert
	found := false
	for _, p := range ps {
		if p.Name == "counter.keys" && p.Group == "counter" && p.Value.(int) == 7 {
			found = true
		}
	}
	if err != nil || !found {
		t.Fatalf("err %v props %+v", err, ps)
	}
}

func TestUnit_GeometryPassTagsEntities(t *testing.T) {
	// Arrange
	s := New("t")
	box := s.Add("box", newCrate())
	cam := view.Default()
	cam.Span = 12
	s.Add("camera", &eye{c: cam})
	get := func() *Scene { return s }
	p := pipeline.New(time.Now, append(pipeline.Standard(), &Geometry{Scene: get}, &Cells{Scene: get})...)
	f := pipeline.NewFrame(24, 10)

	// Act
	Render(p, f, s)
	x, y, _ := f.View.Project(geom.V3(0, 0, 2))
	hit, err := Pick(f, s, int(x)/2, int(y)/4)

	// Assert
	if err != nil || hit != box {
		t.Fatalf("picked %+v err %v", hit, err)
	}
}

func TestUnit_RenderWithoutCameraShowsNotice(t *testing.T) {
	// Arrange
	s := New("t")
	s.Add("box", newCrate())
	get := func() *Scene { return s }
	p := pipeline.New(time.Now, append(pipeline.Standard(), &Geometry{Scene: get}, &Cells{Scene: get})...)
	f := pipeline.NewFrame(24, 10)

	// Act
	Render(p, f, s)
	var text []rune
	for x := 0; x < f.Cells.W; x++ {
		c, _ := f.Cells.At(x, f.Cells.H/2)
		text = append(text, c.Rune)
	}
	_, ok := s.View()

	// Assert
	if ok || !strings.Contains(string(text), "No camera") || !errors.Is(s.SetView(view.Default()), ErrNoCamera) {
		t.Fatalf("view %v row %q", ok, string(text))
	}
}

func TestUnit_CloneIsIndependent(t *testing.T) {
	// Arrange
	s := New("t")
	s.Add("box", newCrate()).At(1, 1, 0)

	// Act
	c, err := Clone(s, registry())
	c.Find("box").Local.Pos.X = 9

	// Assert
	if err != nil || s.Find("box").Local.Pos.X != 1 {
		t.Fatalf("err %v original mutated", err)
	}
}

func TestUnit_GraftGivesFreshIDs(t *testing.T) {
	// Arrange
	s := New("t")
	src := s.Add("box", newCrate())
	src.Add("lid", newCrate())
	var buf bytes.Buffer
	if err := EncodePrefab(&buf, src); err != nil {
		t.Fatal(err)
	}
	copyOf, err := DecodePrefab(&buf, registry())
	if err != nil {
		t.Fatal(err)
	}

	// Act
	got, err := s.Graft(nil, copyOf, -1)

	// Assert
	if err != nil || got.ID == src.ID || got.Children[0].ID == src.Children[0].ID || len(s.Root.Children) != 2 {
		t.Fatalf("err %v ids %d %d", err, got.ID, src.ID)
	}
}

func TestUnit_TweenToStartsNowAndReplaces(t *testing.T) {
	// Arrange
	s := New("t")
	box := s.Add("box", newCrate())
	s.Time = 10

	// Act
	err := s.TweenTo(box, "pos.z", 4.0, 2, anim.Linear)
	_ = s.TweenTo(box, "pos.z", 8.0, 2, anim.Linear)
	s.Time = 11
	errApply := s.ApplyClips()
	var buf bytes.Buffer
	_ = Encode(&buf, s)

	// Assert
	if err != nil || errApply != nil || len(s.Clips) != 1 || math.Abs(box.Local.Pos.Z-4) > 1e-9 {
		t.Fatalf("err %v %v clips %d z %f", err, errApply, len(s.Clips), box.Local.Pos.Z)
	}
	if bytes.Contains(buf.Bytes(), []byte("tween:")) {
		t.Fatal("transient clip was saved")
	}
}

func TestUnit_DisabledBehaviourSkipsAndPersists(t *testing.T) {
	// Arrange
	s := New("t")
	e := s.Add("box", newCrate()).With(&counter{})

	// Act
	err := e.Set("counter.enabled", false)
	_ = s.Update(&Ctx{Scene: s}, .1, []input.Key{'a'})
	var buf bytes.Buffer
	_ = Encode(&buf, s)
	back, errBack := Decode(&buf, registry(), nil)

	// Assert
	c := e.Component("counter").(*counter)
	if err != nil || errBack != nil || c.Ticks != 0 || c.Keys != 0 || back.Find("box").Enabled(back.Find("box").Component("counter")) {
		t.Fatalf("err %v %v counter %+v", err, errBack, c)
	}
}

func instanceScene(t *testing.T) (*Scene, *GameObject) {
	t.Helper()
	src := New("src")
	tower := src.Add("tower", newCrate())
	tower.Add("flag", newCrate()).At(0, 0, 3)
	var pf bytes.Buffer
	if err := EncodePrefab(&pf, tower); err != nil {
		t.Fatal(err)
	}
	s := New("level")
	s.UsePrefabs(prefabMap{"tower": pf.Bytes()})
	inst := s.Add("", nil).At(20, 0, 0)
	inst.Prefab = "tower"
	if err := s.Instantiate(); err != nil {
		t.Fatal(err)
	}
	return s, inst
}

func TestUnit_InstanceTakesPrefabNameWhenUnnamed(t *testing.T) {
	// Arrange
	_, inst := instanceScene(t)

	// Act
	name := inst.Name

	// Assert
	if name != "tower" {
		t.Fatalf("name %q", name)
	}
}

func TestUnit_InstanceTakesPrefabTagUnlessItHasOne(t *testing.T) {
	// Arrange
	src := New("src")
	src.Add("brazier", newCrate()).Tag = "interact"
	var pf bytes.Buffer
	if err := EncodePrefab(&pf, src.Find("brazier")); err != nil {
		t.Fatal(err)
	}
	s := New("level")
	s.UsePrefabs(prefabMap{"brazier": pf.Bytes()})
	plain, own := s.Add("a", nil), s.Add("b", nil)
	plain.Prefab, own.Prefab, own.Tag = "brazier", "brazier", "special"

	// Act
	err := s.Instantiate()

	// Assert
	if err != nil || plain.Tag != "interact" || own.Tag != "special" {
		t.Fatalf("err %v tags %q %q", err, plain.Tag, own.Tag)
	}
}

func TestUnit_OverrideSurvivesRebuildAndSave(t *testing.T) {
	// Arrange
	s, inst := instanceScene(t)
	flag := FindPath(inst, "flag")

	// Act
	err := s.SetProp(flag, "tint", "#ff0000")
	errRoot := s.SetProp(inst, "crate.size", []any{4.0, 4.0, 4.0})
	errPos := s.SetProp(inst, "pos", []any{1.0, 2.0, 0.0})
	_ = s.Instantiate()
	var buf bytes.Buffer
	_ = Encode(&buf, s)
	back, errBack := Decode(&buf, registry(), s.prefabs)

	// Assert
	if err != nil || errRoot != nil || errPos != nil || errBack != nil || len(inst.Overrides) != 2 {
		t.Fatalf("errs %v %v %v %v overrides %+v", err, errRoot, errPos, errBack, inst.Overrides)
	}
	got := back.Find("tower")
	if FindPath(got, "flag").Component("crate").(*crate).Tint != color.RGB(255, 0, 0) || got.Component("crate").(*crate).Size != geom.V3(4, 4, 4) {
		t.Fatalf("overrides lost after reload")
	}
	marked := false
	for _, p := range FindPath(got, "flag").Props() {
		if p.Name == "crate.tint" && p.Overridden {
			marked = true
		}
	}
	if !marked {
		t.Fatal("override not marked in props")
	}
}

func TestUnit_RevertRestoresPrefabValue(t *testing.T) {
	// Arrange
	s, inst := instanceScene(t)
	_ = s.SetProp(FindPath(inst, "flag"), "tint", "#ff0000")

	// Act
	err := s.Revert(FindPath(inst, "flag"), "tint")

	// Assert
	if err != nil || len(inst.Overrides) != 0 || FindPath(inst, "flag").Component("crate").(*crate).Tint == color.RGB(255, 0, 0) {
		t.Fatalf("err %v overrides %+v", err, inst.Overrides)
	}
}

func TestUnit_InheritedChildCannotBeRenamed(t *testing.T) {
	// Arrange
	s, inst := instanceScene(t)

	// Act
	err := s.SetProp(FindPath(inst, "flag"), "name", "banner")

	// Assert
	if !errors.Is(err, ErrPrefabLocked) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_TwoComponentsOfOneKindGetNumberedKeys(t *testing.T) {
	// Arrange
	s := New("t")
	o := s.Add("lamp", newCrate(), newCrate())

	// Act
	err := o.Set("crate#2.tint", "#00ff00")
	short := o.Set("tint", "#ff0000")

	// Assert
	first, second := o.Components[0].(*crate), o.Components[1].(*crate)
	if err != nil || short != nil || second.Tint != color.RGB(0, 255, 0) || first.Tint != color.RGB(255, 0, 0) {
		t.Fatalf("err %v %v first %+v second %+v", err, short, first.Tint, second.Tint)
	}
	if o.Key(o.Components[1]) != "crate#2" {
		t.Fatalf("key %s", o.Key(o.Components[1]))
	}
}

func TestUnit_DecodeVersion1Scene(t *testing.T) {
	// Arrange
	doc := `{"version":1,"name":"old","entities":[{"name":"box","pos":[1,2,3],"scale":2,
		"node":{"kind":"crate","props":{"tint":"#112233"}},
		"behaviours":[{"kind":"counter","off":true,"props":{"keys":5}}]}]}`

	// Act
	s, err := Decode(bytes.NewBufferString(doc), registry(), nil)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	o := s.Find("box")
	if len(o.Components) != 2 || o.Local.Scale != geom.V3(2, 2, 2) || o.Component("crate").(*crate).Tint != color.RGB(17, 34, 51) {
		t.Fatalf("object %+v", o)
	}
	if o.Enabled(o.Component("counter")) || o.Component("counter").(*counter).Keys != 5 {
		t.Fatal("behaviour state lost")
	}
}

func TestUnit_RotationAndScaleComposeThroughParents(t *testing.T) {
	// Arrange
	s := New("t")
	p := s.Add("p").At(10, 0, 0)
	p.Local.Rot = geom.V3(0, 0, 90)
	p.Local.Scale = geom.V3(2, 2, 2)

	// Act
	c := p.Add("c").At(1, 0, 0)

	// Assert
	w := c.World().Pos
	if math.Abs(w.X-10) > 1e-9 || math.Abs(w.Y-2) > 1e-9 {
		t.Fatalf("world %+v", w)
	}
}

type dotted struct{ crate }

func (*dotted) Kind() string { return "pack.crate" }

func TestUnit_DottedKindsResolve(t *testing.T) {
	// Arrange
	s := New("t")
	o := s.Add("thing", &dotted{crate: *newCrate().(*crate)})

	// Act
	err := o.Set("tint", "#00ff00")
	full := o.Set("pack.crate.enabled", false)

	// Assert
	if err != nil || full != nil || o.Components[0].(*dotted).Tint != color.RGB(0, 255, 0) || o.Enabled(o.Components[0]) {
		t.Fatalf("err %v %v", err, full)
	}
}

type link struct{ Target, Box string }

func (*link) Kind() string { return "link" }
func (l *link) bind() *prop.Binder {
	return prop.Bind().Object("target", &l.Target).Component("box", "crate", &l.Box)
}
func (l *link) Props() []prop.Prop        { return l.bind().Props() }
func (l *link) Set(n string, v any) error { return l.bind().Set(n, v) }

func TestUnit_ReferencesSurviveSaveAndLoad(t *testing.T) {
	// Arrange
	s := New("t")
	door := s.Add("door", newCrate())
	lever := s.Add("lever", &link{})
	_ = lever.Set("target", RefOf(lever, door))
	_ = lever.Set("box", CompRefOf(lever, door, door.Components[0]))
	var b bytes.Buffer
	if err := Encode(&b, s); err != nil {
		t.Fatal(err)
	}

	// Act
	back, err := Decode(&b, registry(), nil)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	l := back.Find("lever")
	got, c := back.LookupComponent(l, l.Components[0].(*link).Box, "crate")
	if back.Lookup(l, l.Components[0].(*link).Target) != back.Find("door") || got != back.Find("door") || c == nil {
		t.Fatalf("refs %+v", l.Components[0])
	}
	for _, p := range l.Props() {
		if p.Name == "link.target" && p.Display != "door" {
			t.Fatalf("display %q", p.Display)
		}
	}
}

func TestUnit_ReferencesIntoAndWithinPrefabInstances(t *testing.T) {
	// Arrange
	src := New("src")
	tower := src.Add("tower", newCrate())
	flag := tower.Add("flag", &link{})
	_ = flag.Set("target", RefOf(flag, tower))
	Localize(tower)
	var pf bytes.Buffer
	if err := EncodePrefab(&pf, tower); err != nil {
		t.Fatal(err)
	}
	s := New("level")
	s.UsePrefabs(prefabMap{"tower": pf.Bytes()})
	inst := s.Add("", nil)
	inst.Prefab = "tower"
	if err := s.Instantiate(); err != nil {
		t.Fatal(err)
	}
	outside := s.Add("watcher", &link{})
	innerFlag := FindPath(inst, "flag")

	// Act
	ref := RefOf(outside, innerFlag)
	local := innerFlag.Components[0].(*link).Target

	// Assert
	if s.Lookup(outside, ref) != innerFlag || s.Lookup(innerFlag, local) != inst || local[0] != '~' {
		t.Fatalf("ref %q local %q", ref, local)
	}
}

func TestUnit_UnpackRelinksReferences(t *testing.T) {
	// Arrange
	s, inst := instanceScene(t)
	watcher := s.Add("watcher", &link{})
	_ = watcher.Set("target", RefOf(watcher, FindPath(inst, "flag")))

	// Act
	err := s.Unpack(inst.ID)

	// Assert
	target := s.Lookup(watcher, watcher.Components[0].(*link).Target)
	if err != nil || target == nil || target.Name != "flag" || target.inherited {
		t.Fatalf("err %v target %+v ref %q", err, target, watcher.Components[0].(*link).Target)
	}
}

func TestUnit_GraftRenumbersClashingCopies(t *testing.T) {
	// Arrange
	s := New("t")
	a := s.Add("a", newCrate())
	var b bytes.Buffer
	_ = EncodePrefab(&b, a)
	c, _ := DecodePrefab(&b, registry())

	// Act
	got, err := s.Graft(nil, c, -1)

	// Assert
	if err != nil || got.FID == a.FID || got.FID == 0 {
		t.Fatalf("err %v fids %d %d", err, a.FID, got.FID)
	}
}

func TestUnit_LegacyNamesStillResolve(t *testing.T) {
	// Arrange
	s := New("t")
	door := s.Add("old door", newCrate())

	// Act
	got := s.Lookup(s.Root, "old door")

	// Assert
	if got != door {
		t.Fatalf("got %+v", got)
	}
}

func TestUnit_NewObjectsGetRandomFileIDs(t *testing.T) {
	// Arrange
	a, b := New("a"), New("b")

	// Act
	x, y := a.Add("x"), b.Add("x")

	// Assert
	if x.FID == y.FID || x.FID < 1<<20 || x.FID >= 1<<53 {
		t.Fatalf("fids %d %d", x.FID, y.FID)
	}
}
