package session

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
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
		Packs:     []project.Pack{dragonfall.Pack()},
		Templates: []Template{{Name: "empty"}, {Name: "dragonfall", FS: dragonfall.Template()}},
		Players:   players.Source{FS: fstest.MapFS{"ascdii-linux-amd64": {Data: []byte("PLAYER")}}},
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

func find(s *Session, name string) *scene.Entity {
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
	awake := find(s, "dragon").Node.(*dragonfall.Dragon).Awake
	errUndo := s.Undo()
	undone := find(s, "dragon").Node.(*dragonfall.Dragon).Awake
	errRedo := s.Redo()

	// Assert
	if err != nil || errUndo != nil || errRedo != nil || !awake || undone {
		t.Fatalf("err %v %v %v awake %v undone %v", err, errUndo, errRedo, awake, undone)
	}
	if !find(s, "dragon").Node.(*dragonfall.Dragon).Awake {
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
	picked := s.Pick(int(col), int(row)-1)

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
	z := find(s, "box").Local.Pos.Z
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
	res, err := s.Build(context.Background(), "linux")
	_, errWin := s.Build(context.Background(), "windows")

	// Assert
	if err != nil || len(res.Files) != 1 || filepath.Dir(res.Files[0]) != filepath.Join(dir, "build") {
		t.Fatalf("err %v res %+v", err, res)
	}
	if !errors.Is(errWin, players.ErrNoPlayer) {
		t.Fatalf("got %v", errWin)
	}
}

func TestUnit_DragMovesOnGround(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	id, _ := s.Add(0, "box")
	before := find(s, "box").Local.Pos

	// Act
	err := s.Drag(id, "free", 10, 10, 14, 10)

	// Assert
	after := find(s, "box").Local.Pos
	if err != nil || after.X <= before.X || after.Y != before.Y || after.Z != before.Z {
		t.Fatalf("err %v before %+v after %+v", err, before, after)
	}
}

func TestUnit_RunEmitsFrames(t *testing.T) {
	// Arrange
	s, rec, _ := newSession(t, "")
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
	hour, span := s.edit.Env.Hour, s.edit.Camera.Span
	s.mu.Unlock()
	if err != nil || errCam != nil || hour != 3 || span != 20 {
		t.Fatalf("err %v %v hour %f span %f", err, errCam, hour, span)
	}
}

func TestUnit_EditorCameraLeavesGameCameraAlone(t *testing.T) {
	// Arrange
	s, _, _ := newSession(t, "")
	s.mu.Lock()
	game := s.edit.Camera
	s.mu.Unlock()

	// Act
	s.Pan(5, 3)
	s.Orbit(30, 5)
	s.mu.Lock()
	after, editor := s.edit.Camera, s.cam
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
	before := find(s, "box").Local.Pos

	// Act
	errX := s.Drag(id, "x", 10, 10, 14, 13)
	mid := find(s, "box").Local.Pos
	errZ := s.Drag(id, "z", 10, 10, 10, 8)
	after := find(s, "box").Local.Pos

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
	_ = s.AddBehaviour(id, "script")
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

func childNamed(s *Session, parent uint64, name string) *scene.Entity {
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
		if p.Name == "radius" && p.Overridden {
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
	props, _ := s.Props(find(s, "sprite").ID)
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
