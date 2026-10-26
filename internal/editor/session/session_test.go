package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	stdcolor "image/color"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type recorder struct {
	mu     sync.Mutex
	events map[string]int
}

func (r *recorder) Emit(_ context.Context, event string, _ any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[event]++
}

func (r *recorder) count(e string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.events[e]
}

func newSession(t *testing.T, template string) (*Session, *recorder, string) {
	t.Helper()
	rec := &recorder{events: map[string]int{}}
	s := New(Deps{
		Modules:   []project.Module{dragonfall.Module()},
		Templates: []Template{{Name: "empty"}, {Name: "dragonfall", FS: dragonfall.Template()}},
		Packages:  []pack.Source{dragonfall.Package()},
		Players:   players.Source{Dirs: []fs.FS{fstest.MapFS{"ascdii-player-linux-amd64": {Data: []byte("PLAYER")}}}},
		Emit:      rec,
	})
	dir := filepath.Join(t.TempDir(), "game")
	if _, err := s.Create(context.Background(), dir, "Game", template); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	s.Resize(80, 30)
	return s, rec, dir
}

func find(s *Session, name string) *scene.GameObject {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current().Find(name)
}

func TestUnit_SnapshotEncodesEveryCell(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")

	// Act
	f := s.Snapshot()

	// Assert
	raw, err := base64.StdEncoding.DecodeString(f.Cells)
	if err != nil || f.Cols != 80 || f.Rows != 30 || len(raw) != 80*30*10 || len(f.Passes) < 8 {
		t.Fatalf("err %v frame %dx%d bytes %d", err, f.Cols, f.Rows, len(raw))
	}
}

func TestUnit_SetPropThenUndoRedo(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	dragon := find(s, "dragon")

	// Act
	err := s.SetProp(dragon.ID, "awake", true)
	awake := find(s, "dragon").Component("df.dragon").(*dragonfall.Dragon).Awake
	errUndo := s.Undo()
	undone := find(s, "dragon").Component("df.dragon").(*dragonfall.Dragon).Awake
	errRedo := s.Redo()

	// Assert
	if err != nil || errUndo != nil || errRedo != nil || !awake || undone {
		t.Fatalf("err %v %v %v awake %v undone %v", err, errUndo, errRedo, awake, undone)
	}
	if !find(s, "dragon").Component("df.dragon").(*dragonfall.Dragon).Awake {
		t.Fatal("redo lost the change")
	}
}

func TestUnit_AddSelectsAndPickFindsIt(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")

	// Act
	id, err := s.Add(0, "box")
	f := s.Snapshot()
	col, row := f.Gizmo.Anchor[0], f.Gizmo.Anchor[1]
	picked := s.Pick(int(col), int(row)-1, false)

	// Assert
	if err != nil || s.Selected() != picked || picked != id {
		t.Fatalf("err %v added %d picked %d", err, id, picked)
	}
}

func TestUnit_PlayRunsOnACopy(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	start := find(s, "hero").Local.Pos

	// Act
	err := s.Play()
	_ = s.Key("d")
	for i := 0; i < 20; i++ {
		_, _ = s.Tick(1.0 / 30)
	}
	moved := find(s, "hero").Local.Pos
	s.Stop()

	// Assert
	if err != nil || moved.X <= start.X || find(s, "hero").Local.Pos != start {
		t.Fatalf("err %v start %+v moved %+v after %+v", err, start, moved, find(s, "hero").Local.Pos)
	}
	if !errors.Is(s.Key("d"), ErrNotPlaying) {
		t.Fatal("keys must be rejected outside play mode")
	}
}

func TestUnit_DisablingBrailleChangesOutput(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	before := s.Snapshot().Cells

	// Act
	err := s.SetPassEnabled("braille", false)
	after := s.Snapshot().Cells

	// Assert
	if err != nil || before == after {
		t.Fatalf("err %v unchanged %v", err, before == after)
	}
}

func TestUnit_AddKeyAnimatesOnScrub(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	s.SetLive(false)

	// Act
	err1 := s.AddKey(id, "pos.z", 0)
	_ = s.SetProp(id, "pos.z", 4.0)
	err2 := s.AddKey(id, "pos.z", 2)
	err3 := s.SetTime(1)
	clips, _ := s.Clips(id)

	// Assert
	z := find(s, "Cube").Local.Pos.Z
	if err1 != nil || err2 != nil || err3 != nil || len(clips) != 1 || z <= 0 || z >= 4 {
		t.Fatalf("errs %v %v %v clips %d z %f", err1, err2, err3, len(clips), z)
	}
}

func TestUnit_SavePrefabMakesInstance(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	parent, _ := s.Add(0, "empty")
	_, _ = s.Add(parent, "cylinder")
	ctx := context.Background()

	// Act
	err := s.SavePrefab(ctx, parent, "prefabs/thing")
	copyID, errDup := s.Duplicate(parent)

	// Assert
	if _, statErr := os.Stat(filepath.Join(dir, "prefabs", "thing.prefab.json")); err != nil || statErr != nil {
		t.Fatalf("err %v stat %v", err, statErr)
	}
	tree := s.Tree()
	var inst, dup *Node
	for i := range tree {
		if tree[i].ID == parent {
			inst = &tree[i]
		}
		if tree[i].ID == copyID {
			dup = &tree[i]
		}
	}
	if errDup != nil || inst == nil || inst.Prefab == "" || len(inst.Children) != 1 || !inst.Children[0].Inherited || dup == nil || len(dup.Children) != 1 {
		t.Fatalf("errDup %v inst %+v dup %+v", errDup, inst, dup)
	}
}

func TestUnit_SaveThenReopenKeepsEdits(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	id, _ := s.Add(0, "box")
	_ = s.SetProp(id, "name", "crate two")
	ctx := context.Background()

	// Act
	err := s.Save(ctx)
	info, _ := s.Info(ctx)
	errOpen := s.Open(ctx, info.Manifest)

	// Assert
	if err != nil || errOpen != nil || find(s, "crate two") == nil || info.Dir != dir {
		t.Fatalf("err %v %v", err, errOpen)
	}
}

func TestUnit_BuildBundlesProject(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")

	// Act
	res, err := s.Build(context.Background(), "linux", BuildOptions{Version: "0.2.0"})
	one, errOne := s.Build(context.Background(), "linux", BuildOptions{Single: true})
	_, errWin := s.Build(context.Background(), "windows", BuildOptions{})

	// Assert
	if err != nil || len(res.Files) != 1 || res.Dir != filepath.Join(dir, "build", "linux", "Game") {
		t.Fatalf("err %v res %+v", err, res)
	}
	if _, err := os.Stat(filepath.Join(res.Dir, "data", "game.pak")); err != nil {
		t.Fatal("the folder build has no packed data")
	}
	if s.proj.Manifest.Version != "0.2.0" {
		t.Fatalf("version %s", s.proj.Manifest.Version)
	}
	if errOne != nil || filepath.Dir(one.Files[0]) != filepath.Join(dir, "build") {
		t.Fatalf("single err %v res %+v", errOne, one)
	}
	if !errors.Is(errWin, players.ErrNoPlayer) {
		t.Fatalf("got %v", errWin)
	}
}

func TestUnit_DragMovesOnGround(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	before := find(s, "Cube").Local.Pos

	// Act
	err := s.Drag(id, "free", 10, 10, 14, 10)

	// Assert
	after := find(s, "Cube").Local.Pos
	if err != nil || after.X <= before.X || after.Y != before.Y || after.Z != before.Z {
		t.Fatalf("err %v before %+v after %+v", err, before, after)
	}
}

func TestUnit_TurnAndStretchChangeTransform(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")

	// Act
	errTurn := s.Turn(id, "x", 10, 10, 20, 10)
	errGrow := s.Stretch(id, "free", 10, 10, 20, 10)
	errAxis := s.Stretch(id, "w", 10, 10, 20, 10)

	// Assert
	l := find(s, "Cube").Local
	if errTurn != nil || errGrow != nil || l.Rot.X != 30 || l.Scale.X <= 1 || l.Scale.X != l.Scale.Z || !errors.Is(errAxis, ErrBadAxis) {
		t.Fatalf("turn %v grow %v axis %v local %+v", errTurn, errGrow, errAxis, l)
	}
}

func TestUnit_RawFrameHasHeaderAndCells(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")

	// Act
	f := s.Snapshot()
	raw := s.Raw()

	// Assert
	if len(raw) != 12+80*30*10 || raw[0] != 80 || raw[4] != 30 || uint64(raw[8]) != f.Seq%256 {
		t.Fatalf("raw %d bytes header %v seq %d", len(raw), raw[:12], f.Seq)
	}
}

func TestUnit_RunEmitsFrames(t *testing.T) {
	// Arrange
	s, rec, _ := newSession(t, "")
	s.SetLive(true)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	// Act
	err := s.Run(ctx, 60)

	// Assert
	if err != nil || rec.count(FrameEvent) < 3 {
		t.Fatalf("err %v frames %d", err, rec.count(FrameEvent))
	}
}

func TestUnit_SceneSettingsAreEditable(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")

	// Act
	s.SetView(GameView)
	err := s.SetSceneProp("env.hour", 3)
	errCam := s.SetSceneProp("camera.span", 20)

	// Assert
	s.mu.Lock()
	hour := s.edit.Env.Hour
	s.mu.Unlock()
	if err != nil || hour != 3 || !errors.Is(errCam, prop.ErrUnknownProp) {
		t.Fatalf("err %v hour %f camera %v", err, hour, errCam)
	}
}

func TestUnit_EditorCameraLeavesGameCameraAlone(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	s.mu.Lock()
	game, _ := s.edit.View()
	s.mu.Unlock()

	// Act
	s.Pan(5, 3)
	s.Orbit(30, 5)
	s.mu.Lock()
	after, _ := s.edit.View()
	editor := s.cam
	s.mu.Unlock()

	// Assert
	if after != game || editor == game {
		t.Fatalf("game camera moved: %+v", after)
	}
}

func TestUnit_DropPlacesPrefabUnderCursor(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")

	// Act
	id, err := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)
	f := s.Snapshot()

	// Assert
	if err != nil || f.Gizmo == nil || f.Gizmo.ID != id || len(f.Gizmo.Axes) != 3 {
		t.Fatalf("err %v gizmo %+v", err, f.Gizmo)
	}
	if dx, dy := f.Gizmo.Anchor[0]-40, f.Gizmo.Anchor[1]-15; dx*dx+dy*dy > 4 {
		t.Fatalf("dropped at %v", f.Gizmo.Anchor)
	}
}

func TestUnit_DragAxisConstrains(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	before := find(s, "Cube").Local.Pos

	// Act
	errX := s.Drag(id, "x", 10, 10, 14, 13)
	mid := find(s, "Cube").Local.Pos
	errZ := s.Drag(id, "z", 10, 10, 10, 8)
	after := find(s, "Cube").Local.Pos

	// Assert
	if errX != nil || errZ != nil || mid.X <= before.X || mid.Y != before.Y || after.Z <= 0 || after.X != mid.X {
		t.Fatalf("before %+v mid %+v after %+v", before, mid, after)
	}
}

func TestUnit_ScriptLogsReachConsole(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	ctx := context.Background()
	rel, err := s.NewScript(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := s.Add(0, "box")
	_, _ = s.AddComponent(id, "script")
	_ = s.SetProp(id, "script.file", rel)
	abs, _ := s.AbsPath(rel)
	_ = os.WriteFile(abs, []byte(`function update(self) log("hi", 1 + 1) end`), 0o644)
	s.reloadFiles(ctx, s.dir, []string{rel})

	// Act
	_ = s.Play()
	_, _ = s.Tick(1.0 / 30)

	// Assert
	found := false
	for _, l := range s.Logs() {
		if l.Text == "hi 2" && l.Source == rel {
			found = true
		}
	}
	if !found {
		t.Fatalf("logs %+v", s.Logs())
	}
}

func TestUnit_StepAdvancesPausedGame(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	_ = s.Play()
	s.Pause()
	s.mu.Lock()
	t0 := s.play.Time
	s.mu.Unlock()

	// Act
	s.Step()
	_, _ = s.Tick(.5)
	_, _ = s.Tick(.5)

	// Assert
	s.mu.Lock()
	dt := s.play.Time - t0
	s.mu.Unlock()
	if dt < .015 || dt > .02 {
		t.Fatalf("advanced %f", dt)
	}
}

func TestUnit_CloseProjectReturnsToStart(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	ctx := context.Background()

	// Act
	s.CloseProject()
	info, err := s.Info(ctx)
	_, errAdd := s.Assets(ctx)

	// Assert
	if err != nil || info.Open || !errors.Is(errAdd, ErrNoProject) || len(s.Tree()) != 0 {
		t.Fatalf("info %+v err %v add %v", info, err, errAdd)
	}
}

func childNamed(s *Session, parent uint64, name string) *scene.GameObject {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, _ := s.edit.ByID(parent)
	return scene.FindPath(p, name)
}

func TestUnit_InstanceEditBecomesOverrideThenApply(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "dragonfall")
	id, _ := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)
	glow := childNamed(s, id, "glow")
	ctx := context.Background()

	// Act
	err := s.SetProp(glow.ID, "radius", 20)
	props, _ := s.Props(glow.ID)
	errApply := s.Apply(ctx, id, "")
	raw, _ := os.ReadFile(filepath.Join(dir, "prefabs", "lantern.prefab.json"))

	// Assert
	marked := false
	for _, p := range props {
		if p.Name == "light.radius" && p.Overridden {
			marked = true
		}
	}
	if err != nil || errApply != nil || !marked || !strings.Contains(string(raw), `"radius": 20`) {
		t.Fatalf("err %v %v marked %v", err, errApply, marked)
	}
	if root := find(s, "lantern 1"); root.Prefab == "" {
		t.Fatal("template lantern lost its prefab")
	}
}

func TestUnit_RevertDropsOverride(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	id, _ := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)
	_ = s.SetProp(childNamed(s, id, "glow").ID, "radius", 20)

	// Act
	err := s.Revert(id, "")

	// Assert
	glow := childNamed(s, id, "glow")
	props, _ := s.Props(glow.ID)
	for _, p := range props {
		if p.Name == "radius" && (p.Overridden || p.Value.(float64) == 20) {
			t.Fatalf("still overridden: %+v", p)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnit_RenameAssetKeepsReferences(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "dragonfall")
	ctx := context.Background()

	// Act
	to, err := s.RenameAsset(ctx, "prefabs/lantern.prefab.json", "street-lamp")
	_ = s.Save(ctx)
	info, _ := s.Info(ctx)
	errOpen := s.Open(ctx, info.Manifest)

	// Assert
	if err != nil || errOpen != nil || to != "prefabs/street-lamp.prefab.json" {
		t.Fatalf("err %v %v to %s", err, errOpen, to)
	}
	if _, err := os.Stat(filepath.Join(dir, "prefabs", "street-lamp.prefab.json.meta")); err != nil {
		t.Fatal("meta did not move")
	}
	if l := find(s, "lantern 1"); l == nil || len(l.Children) == 0 {
		t.Fatal("instance broke after rename")
	}
}

func TestUnit_OpenMigratesPathReferences(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	ctx := context.Background()
	art := filepath.Join(dir, "art", "a.json")
	_ = os.MkdirAll(filepath.Dir(art), 0o755)
	_ = os.WriteFile(art, []byte(`{"name":"a","pivot":[0,0],"palette":{"x":{"glyph":"X","fg":"#ffffff","bg":"#000000"}},"rows":["x"]}`), 0o644)
	_ = s.Reload(ctx)
	id, _ := s.Add(0, "sprite")
	s.mu.Lock()
	e, _ := s.edit.ByID(id)
	_ = e.Set("asset", "art/a.json")
	s.mu.Unlock()
	_ = s.Save(ctx)
	info, _ := s.Info(ctx)

	// Act
	err := s.Open(ctx, info.Manifest)

	// Assert
	props, _ := s.Props(find(s, "Sprite").ID)
	for _, p := range props {
		if p.Name == "asset" && !guidValid(p.Value) {
			t.Fatalf("not migrated: %v", p.Value)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

func guidValid(v any) bool {
	str, _ := v.(string)
	return guid.Valid(str)
}

func TestUnit_ComponentsAddMoveResetRemove(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "light")

	// Act
	key, errAdd := s.AddComponent(id, "light")
	errSet := s.SetProp(id, key+".radius", 30)
	errMove := s.MoveComponent(id, 1, 0)
	errReset := s.ResetComponent(id, 0)
	errRemove := s.RemoveComponent(id, 1)

	// Assert
	o := find(s, "Point Light")
	if errAdd != nil || errSet != nil || errMove != nil || errReset != nil || errRemove != nil {
		t.Fatalf("errs %v %v %v %v %v", errAdd, errSet, errMove, errReset, errRemove)
	}
	if key != "light#2" || len(o.Components) != 1 || o.Components[0].(*components.Light).Radius != 8 {
		t.Fatalf("key %s components %+v", key, o.Components)
	}
}

func TestUnit_PrefabInstanceComponentsAreLocked(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	id, _ := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)

	// Act
	_, err := s.AddComponent(id, "light")

	// Assert
	if !errors.Is(err, scene.ErrPrefabLocked) {
		t.Fatalf("err %v", err)
	}
}

func TestUnit_ImportModelBringsMaterialsAndDrops(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	src := t.TempDir()
	files := map[string]string{
		"hut.obj": "mtllib hut.mtl\nv 0 0 0\nv 2 0 0\nv 2 2 0\nusemtl roof\nf 1 2 3\n",
		"hut.mtl": "newmtl roof\nKd 1 0 0\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(src, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Act
	got, err := s.Import(context.Background(), []string{filepath.Join(src, "hut.obj")}, "models")
	again, errAgain := s.Import(context.Background(), []string{filepath.Join(src, "hut.obj")}, "models")
	id, errDrop := s.Drop("model:models/hut.obj", 40, 15)
	_, errBad := s.Import(context.Background(), []string{filepath.Join(src, "hut.mtl")}, "")

	// Assert
	if err != nil || errAgain != nil || errDrop != nil || len(got) != 1 || got[0] != "models/hut.obj" || again[0] != "models/hut 2.obj" {
		t.Fatalf("errs %v %v %v got %v again %v", err, errAgain, errDrop, got, again)
	}
	for _, f := range []string{"models/hut.obj.meta", "models/hut.mtl"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	if !errors.Is(errBad, ErrUnsupported) {
		t.Fatalf("mtl import err %v", errBad)
	}
	s.mu.Lock()
	o, _ := s.edit.ByID(id)
	s.mu.Unlock()
	if o == nil || o.Name != "hut" || o.Component("mesh") == nil {
		t.Fatalf("dropped %+v", o)
	}
}

func TestUnit_SetRefPointsFieldsAtObjectsAndComponents(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	lamp, _ := s.Add(0, "light")
	sw, _ := s.Add(0, "empty")
	_, _ = s.AddComponent(sw, "toggle")

	// Act
	err := s.SetRef(sw, "target", lamp, "")
	props, _ := s.Props(sw)
	errWrong := s.SetRef(sw, "visible", lamp, "")
	dup, errDup := s.Duplicate(sw)

	// Assert
	shown := ""
	for _, p := range props {
		if p.Name == "toggle.target" {
			shown = p.Display
		}
	}
	if err != nil || shown != "Point Light" || !errors.Is(errWrong, ErrNotReference) || errDup != nil || dup == 0 {
		t.Fatalf("err %v shown %q wrong %v dup %v", err, shown, errWrong, errDup)
	}
}

func TestUnit_DuplicateRemapsInternalReferences(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	root, _ := s.Add(0, "empty")
	lamp, _ := s.Add(root, "light")
	_, _ = s.AddComponent(root, "toggle")
	_ = s.SetRef(root, "target", lamp, "")

	// Act
	dup, err := s.Duplicate(root)

	// Assert
	s.mu.Lock()
	defer s.mu.Unlock()
	c, _ := s.edit.ByID(dup)
	got := s.edit.Lookup(c, c.Component("toggle").(*components.Toggle).Target)
	if err != nil || got == nil || got.Parent() != c {
		t.Fatalf("err %v target %+v", err, got)
	}
}

func TestUnit_MaterialTextureAndSpriteFromImportedPicture(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	ctx := context.Background()
	src := filepath.Join(t.TempDir(), "brick.png")
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < 64; i++ {
		img.Set(i%8, i/8, stdcolor.NRGBA{R: 180, G: 70, B: 50, A: 255})
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	_ = os.WriteFile(src, b.Bytes(), 0o644)

	// Act
	tex, errImp := s.Import(ctx, []string{src}, "")
	mat, errMat := s.CreateMaterial(ctx, "brick")
	errSet := s.SetAssetProp(ctx, mat, "texture", tex[0])
	errTex := s.SetAssetProp(ctx, tex[0], "filter", "nearest")
	art, errArt := s.ConvertToSprite(ctx, tex[0], 6)
	thumb, errThumb := s.Thumb(tex[0], 32)
	id, _ := s.Add(0, "box")
	errBox := s.SetProp(id, "material", mat)

	// Assert
	for _, err := range []error{errImp, errMat, errSet, errTex, errArt, errThumb, errBox} {
		if err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mat)))
	meta, _ := os.ReadFile(filepath.Join(dir, "brick.png.meta"))
	if tex[0] != "brick.png" || !strings.Contains(string(raw), `"texture"`) || !strings.Contains(string(meta), "nearest") {
		t.Fatalf("tex %v material %s meta %s", tex, raw, meta)
	}
	if art != "brick.sprite.json" || !strings.HasPrefix(thumb, "data:image/png;base64,") {
		t.Fatalf("art %s thumb %.30s", art, thumb)
	}
	if _, ok := s.lib.DrawMaterial(mat); !ok {
		t.Fatal("material not loaded")
	}
}

func TestUnit_FoldersMoveAssetsAndKeepReferences(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "dragonfall")
	ctx := context.Background()
	id, _ := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)

	// Act
	folder, errMk := s.CreateFolder(ctx, "props/street")
	moved, errMv := s.Move(ctx, "prefabs/lantern.prefab.json", folder)
	renamed, errRn := s.RenameFolder(ctx, "props", "decor")
	errInto := func() error { _, err := s.Move(ctx, "decor", "decor/street"); return err }()
	folders, _ := s.Folders(ctx)
	errReload := s.Reload(ctx)

	// Assert
	for _, err := range []error{errMk, errMv, errRn, errReload} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if moved != "props/street/lantern.prefab.json" || renamed != "decor" || !errors.Is(errInto, ErrBadPath) {
		t.Fatalf("moved %s renamed %s into %v", moved, renamed, errInto)
	}
	if _, err := os.Stat(filepath.Join(dir, "decor/street/lantern.prefab.json.meta")); err != nil {
		t.Fatal("meta did not move along")
	}
	s.mu.Lock()
	inst, _ := s.edit.ByID(id)
	kids := len(inst.Children)
	s.mu.Unlock()
	if kids == 0 || !strings.Contains(strings.Join(folders, ","), "decor/street") {
		t.Fatalf("instance lost its prefab (%d children) folders %v", kids, folders)
	}
}

func TestUnit_DeleteForgetsAssets(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	ctx := context.Background()
	rel, _ := s.CreateMaterial(ctx, "old")

	// Act
	err := s.Delete(ctx, rel)
	errOut := s.Delete(ctx, "")

	// Assert
	if _, statErr := os.Stat(filepath.Join(dir, rel+".meta")); err != nil || statErr == nil {
		t.Fatalf("err %v meta still there", err)
	}
	if _, ok := s.lib.DrawMaterial(rel); ok || !errors.Is(errOut, ErrBadPath) {
		t.Fatalf("material still known, deleting the project err %v", errOut)
	}
}

func TestUnit_ImportedFolderKeepsGUIDsFromItsMetas(t *testing.T) {
	// Arrange
	src, _, from := newSession(t, "dragonfall")
	pkg := filepath.Join(t.TempDir(), "street")
	for _, f := range []string{"prefabs/lantern.prefab.json", "prefabs/lantern.prefab.json.meta", "scripts/crystal.lua", "scripts/crystal.lua.meta"} {
		b, err := os.ReadFile(filepath.Join(from, f))
		if err != nil {
			t.Fatal(err)
		}
		_ = os.MkdirAll(filepath.Dir(filepath.Join(pkg, f)), 0o755)
		_ = os.WriteFile(filepath.Join(pkg, f), b, 0o644)
	}
	_ = os.WriteFile(filepath.Join(pkg, "notes.txt"), []byte("hi"), 0o644)
	want := src.lib.Ref("prefabs/lantern.prefab.json")
	s, _, dir := newSession(t, "")
	ctx := context.Background()

	// Act
	got, err := s.Import(ctx, []string{pkg}, "")
	again, errAgain := s.Import(ctx, []string{pkg}, "")
	id, errDrop := s.Drop("prefab:"+want, 30, 10)

	// Assert
	if err != nil || errAgain != nil || errDrop != nil || got[0] != "street" || again[0] != "street 2" {
		t.Fatalf("errs %v %v %v got %v again %v", err, errAgain, errDrop, got, again)
	}
	if s.lib.Resolve(want) != "street/prefabs/lantern.prefab.json" || s.lib.Ref("street 2/prefabs/lantern.prefab.json") == want {
		t.Fatalf("guid %s resolves to %s", want, s.lib.Resolve(want))
	}
	if _, err := os.Stat(filepath.Join(dir, "street", "notes.txt")); err != nil {
		t.Fatal("plain files should come along")
	}
	s.mu.Lock()
	inst, _ := s.edit.ByID(id)
	s.mu.Unlock()
	if len(inst.Children) == 0 {
		t.Fatal("instance of the imported prefab is empty")
	}
}

func TestUnit_PackageInstallUseAndRemove(t *testing.T) {
	// Arrange
	s, _, dir := newSession(t, "")
	ctx := context.Background()
	before, _ := s.PackageList(ctx)
	hasCastle := func() bool {
		return slices.Contains(s.Kinds().Components, Kind{Kind: "df.castle", Meta: s.reg.Meta("df.castle")})
	}
	castleBefore := hasCastle()

	// Act
	home, errInstall := s.InstallPackage(ctx, "builtin:dragonfall")
	list, _ := s.PackageList(ctx)
	castleAfter := hasCastle()
	id, errDrop := s.Drop("prefab:packages/dragonfall/prefabs/lantern.prefab.json", 30, 10)
	errMove := func() error { _, err := s.Move(ctx, "packages/dragonfall/prefabs/lantern.prefab.json", ""); return err }()
	errDelete := s.Delete(ctx, "packages/dragonfall/scripts/crystal.lua")
	_ = s.Remove(id)
	errRemove := s.RemovePackage(ctx, "dragonfall")

	// Assert
	if castleBefore || len(before.Installed) != 0 || len(before.Available) != 1 {
		t.Fatalf("a new project should start without the sample: %v %+v", castleBefore, before)
	}
	if errInstall != nil || errDrop != nil || errRemove != nil || home != "packages/dragonfall" || !castleAfter || len(list.Installed) != 1 {
		t.Fatalf("errs %v %v %v castle %v list %+v", errInstall, errDrop, errRemove, castleAfter, list)
	}
	if !errors.Is(errMove, ErrPackageLocked) || !errors.Is(errDelete, ErrPackageLocked) {
		t.Fatalf("package edits allowed: %v %v", errMove, errDelete)
	}
	if hasCastle() || slices.Contains(s.proj.Manifest.Modules, "dragonfall") {
		t.Fatal("module should leave with its package")
	}
	if _, err := os.Stat(filepath.Join(dir, "packages", "dragonfall")); err == nil {
		t.Fatal("package folder still there")
	}
}

func TestUnit_PackageClashAndExport(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	ctx := context.Background()
	out := filepath.Join(t.TempDir(), "sample")

	// Act
	_, clash := s.InstallPackage(ctx, "builtin:dragonfall")
	errExport := s.ExportPackage(ctx, "prefabs", pack.Manifest{Name: "lanterns", Version: "1.0.0"}, out)
	src, closer, errOpen := pack.Open(out + pack.Ext)

	// Assert
	if !errors.Is(clash, pack.ErrConflict) || errExport != nil || errOpen != nil {
		t.Fatalf("clash %v export %v open %v", clash, errExport, errOpen)
	}
	defer closer.Close()
	files, _ := pack.Files(src.FS)
	if src.Manifest.Name != "lanterns" || !slices.Contains(files, "lantern.prefab.json.meta") {
		t.Fatalf("manifest %+v files %v", src.Manifest, files)
	}
}

func TestUnit_ClipsAndControllerForAnAnimator(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	ctx := context.Background()
	id, _ := s.Add(0, "box")

	// Act
	errIdle := s.NewClip(id, "idle")
	_ = s.AddKey(id, "pos.z", 0)
	errRun := s.NewClip(id, "run")
	_ = s.AddKey(id, "pos.z", 1)
	clips, _ := s.Clips(id)
	errRename := s.RenameClip(clips[1].Index, "walk")
	rel, errCtrl := s.CreateController(ctx, "box")
	c, _ := s.Controller(rel)
	c.States = append(c.States, animator.State{Name: "walk", Clip: "walk"})
	errSave := s.SetController(ctx, rel, c)
	back, _ := s.Controller(rel)
	after, _ := s.Clips(id)

	// Assert
	if errIdle != nil || errRun != nil || errRename != nil || errCtrl != nil || errSave != nil {
		t.Fatalf("errs %v %v %v %v %v", errIdle, errRun, errRename, errCtrl, errSave)
	}
	if len(after) != 2 || after[1].Name != "walk" || len(after[0].Tracks) != 1 || len(after[1].Tracks) != 1 || !after[1].Active {
		t.Fatalf("clips %+v", after)
	}
	if rel != "box.animator.json" || len(back.States) != 2 {
		t.Fatalf("controller %s %+v", rel, back)
	}
}

func TestUnit_PlayModeSoundsBecomeEvents(t *testing.T) {
	// Arrange
	s, rec, dir := newSession(t, "")
	_ = os.WriteFile(filepath.Join(dir, "beep.wav"), []byte("RIFF"), 0o644)
	id, _ := s.Add(0, "empty")
	_, _ = s.AddComponent(id, "audio_source")
	_ = s.SetProp(id, "audio_source.clip", "beep.wav")

	// Act
	errPlay := s.Play()
	_, _ = s.Tick(1.0 / 60)
	s.Stop()
	file, errFile := s.AssetFile("beep.wav")
	_, errOut := s.AssetFile("../secret")

	// Assert
	if errPlay != nil || rec.count("audio") < 2 || errFile != nil || filepath.Base(file) != "beep.wav" {
		t.Fatalf("play %v events %d file %v %s", errPlay, rec.count("audio"), errFile, file)
	}
	if errOut == nil && !strings.HasPrefix(file, dir) {
		t.Fatal("paths must stay inside the project")
	}
}

func TestUnit_BrushPaintsTheSelectedObject(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	_ = s.SetProp(id, "box.size", []float64{6, 6, 6})
	s.mu.Lock()
	s.cam.Target, s.cam.Span = s.spawnPoint(), 12
	s.mu.Unlock()
	s.Resize(40, 20)
	_, _ = s.Tick(0)
	hit := func() (float64, float64) {
		s.mu.Lock()
		defer s.mu.Unlock()
		b := s.frame.Buf
		for y := 0; y < b.H; y++ {
			for x := 0; x < b.W; x++ {
				if b.At(x, y).Tag == uint32(id) {
					return (float64(x) + .5) / raster.SubW, (float64(y) + .5) / raster.SubH
				}
			}
		}
		return -1, -1
	}
	col, row := hit()

	// Act
	err := s.PaintAt(id, col, row, "#ff0000", 1, false)
	missed := s.PaintAt(id, 0, 0, "#ff0000", 1, false)

	// Assert
	p, _ := find(s, "Cube").Component("paint").(*components.Paint)
	if err != nil || p == nil || len(p.Strokes) != 1 || !errors.Is(missed, ErrMissed) {
		t.Fatalf("err %v missed %v paint %v at %.1f,%.1f", err, missed, p, col, row)
	}
}

func TestUnit_PrefabModeEditsAndUpdatesInstances(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "dragonfall")
	ctx := context.Background()
	id, _ := s.Drop("prefab:prefabs/lantern.prefab.json", 40, 15)
	_, _ = s.Add(0, "box")
	unsavedBefore := s.unsaved

	// Act
	errOpen := s.OpenPrefab("prefabs/lantern.prefab.json")
	glow := find(s, "glow")
	errSet := s.SetProp(glow.ID, "light.radius", 30)
	errSave := s.Save(ctx)
	info, _ := s.Info(ctx)
	errClose := s.ClosePrefab()

	// Assert
	if errOpen != nil || errSet != nil || errSave != nil || errClose != nil || info.Prefab != "prefabs/lantern.prefab.json" {
		t.Fatalf("errs %v %v %v %v info %+v", errOpen, errSet, errSave, errClose, info)
	}
	if !s.unsaved || !unsavedBefore || find(s, "Cube") == nil {
		t.Fatal("the scene should come back with its unsaved edits")
	}
	got := childNamed(s, id, "glow").Component("light").(*components.Light).Radius
	if got != 30 {
		t.Fatalf("instance radius %v", got)
	}
}

func TestUnit_ParticlesPresetAndBurstInEditor(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "emitter")

	// Act
	_, errPreset := s.Particles(id, "emitter", "preset", "explosion")
	n, errEmit := s.Particles(id, "emitter", "emit", "")
	_, errBad := s.Particles(id, "emitter", "preset", "nope")
	_, errKind := s.Particles(id, "nothing", "play", "")

	// Assert
	if errPreset != nil || errEmit != nil || n < 120 || !errors.Is(errBad, ErrUnknownPreset) || !errors.Is(errKind, ErrNotParticles) {
		t.Fatalf("preset %v emit %v n %d bad %v kind %v", errPreset, errEmit, n, errBad, errKind)
	}
}

func TestUnit_ColliderHandlesShowAndDragResizesOneSide(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	key, errAdd := s.AddComponent(id, "box_collider")
	s.Select(id)
	f := s.Snapshot()

	// Act
	err := s.DragCollider(id, key, "x+", 10, 10, 14, 10)
	errBad := s.DragCollider(id, key, "w", 10, 10, 14, 10)

	// Assert
	col := find(s, "Cube").ComponentByKey(key).(*components.BoxCollider)
	if errAdd != nil || err != nil || f.Gizmo == nil || len(f.Gizmo.Handles) != 5 || col.Size.X <= 2 || col.Offset.X <= 0 || !errors.Is(errBad, ErrBadAxis) {
		t.Fatalf("add %v err %v handles %v collider %+v bad %v", errAdd, err, f.Gizmo, col, errBad)
	}
	if left := col.Offset.X - col.Size.X/2; math.Abs(left+1) > 1e-9 {
		t.Fatalf("the left side moved to %f", left)
	}
}

func TestUnit_DragMovesAChildInItsParentsSpace(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	parent, _ := s.Add(0, "box")
	child, _ := s.Add(parent, "blob")
	errSet := s.SetProp(parent, "rot", []any{0.0, 0.0, 90.0})
	sc := s.current()
	c, _ := sc.ByID(child)
	before, worldBefore := c.Local.Pos, c.World().Pos

	// Act
	err := s.Drag(child, "free", 10, 10, 14, 10)

	// Assert
	after, worldAfter := c.Local.Pos, c.World().Pos
	if errSet != nil || err != nil || after == before || worldAfter.X <= worldBefore.X || math.Abs(worldAfter.Y-worldBefore.Y) > 1e-6 {
		t.Fatalf("err %v %v local %+v -> %+v world %+v -> %+v", errSet, err, before, after, worldBefore, worldAfter)
	}
}
