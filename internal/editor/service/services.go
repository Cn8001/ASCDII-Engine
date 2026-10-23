package service

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/session"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
)

type ProjectService struct{ d Deps }

func (p *ProjectService) Info() (session.Info, error) { return wrap1(p.d.Session.Info(p.d.Ctx.Get())) }

func (p *ProjectService) Recent() ([]string, error) { return wrap1(p.d.Recent.Load(p.d.Ctx.Get())) }

func (p *ProjectService) Open(path string) (session.Info, error) {
	ctx := p.d.Ctx.Get()
	if err := p.d.Session.Open(ctx, path); err != nil {
		return session.Info{}, wrap(err)
	}
	if _, err := p.d.Recent.Add(ctx, path); err != nil {
		return session.Info{}, wrap(err)
	}
	return p.Info()
}

func (p *ProjectService) OpenDialog() (session.Info, error) {
	ctx := p.d.Ctx.Get()
	path, err := p.d.Dialogs.OpenFile(ctx, "Open project", []Filter{{Name: "ASCDII project", Pattern: "*" + project.Ext}})
	if err != nil {
		return session.Info{}, wrap(err)
	}
	if path == "" {
		return session.Info{}, wrap(errCancelled)
	}
	return p.Open(path)
}

func (p *ProjectService) PickFolder() (string, error) {
	dir, err := p.d.Dialogs.PickDir(p.d.Ctx.Get(), "Project location")
	if err == nil && dir == "" {
		err = errCancelled
	}
	return dir, wrap(err)
}

// Create makes <parent>/<name> from a template and opens it.
func (p *ProjectService) Create(parent, name, template string) (session.Info, error) {
	if template == "empty" {
		template = ""
	}
	manifest, err := p.d.Session.Create(p.d.Ctx.Get(), filepath.Join(parent, project.FileName(name)), name, template)
	if err != nil {
		return session.Info{}, wrap(err)
	}
	if _, err := p.d.Recent.Add(p.d.Ctx.Get(), manifest); err != nil {
		return session.Info{}, wrap(err)
	}
	return p.Info()
}

func (p *ProjectService) Save() error { return wrap(p.d.Session.Save(p.d.Ctx.Get())) }
func (p *ProjectService) SaveAs(rel string) error {
	return wrap(p.d.Session.SaveAs(p.d.Ctx.Get(), rel))
}
func (p *ProjectService) NewScene(rel string) error {
	return wrap(p.d.Session.NewScene(p.d.Ctx.Get(), rel))
}
func (p *ProjectService) OpenScene(rel string) error {
	return wrap(p.d.Session.OpenScene(p.d.Ctx.Get(), rel))
}
func (p *ProjectService) SetStart() error { return wrap(p.d.Session.SetStart(p.d.Ctx.Get())) }
func (p *ProjectService) Settings() (session.ProjectSettings, error) {
	return wrap1(p.d.Session.ProjectSettings())
}
func (p *ProjectService) SetSettings(v session.ProjectSettings) error {
	return wrap(p.d.Session.SetProjectSettings(p.d.Ctx.Get(), v))
}
func (p *ProjectService) Close()      { p.d.Session.CloseProject() }
func (p *ProjectService) Undo() error { return wrap(p.d.Session.Undo()) }
func (p *ProjectService) Redo() error { return wrap(p.d.Session.Redo()) }

type SceneService struct{ d Deps }

func (s *SceneService) Tree() []session.Node    { return s.d.Session.Tree() }
func (s *SceneService) Kinds() session.Kinds    { return s.d.Session.Kinds() }
func (s *SceneService) Select(id uint64)        { s.d.Session.Select(id) }
func (s *SceneService) Selected() uint64        { return s.d.Session.Selected() }
func (s *SceneService) SceneProps() []prop.Prop { return s.d.Session.SceneProps() }

func (s *SceneService) Props(id uint64) ([]prop.Prop, error) { return wrap1(s.d.Session.Props(id)) }
func (s *SceneService) SetProp(id uint64, name string, value any) error {
	return wrap(s.d.Session.SetProp(id, name, value))
}

// SetRef points a reference field at an object, or a component of it by key.
func (s *SceneService) SetRef(id uint64, name string, target uint64, key string) error {
	return wrap(s.d.Session.SetRef(id, name, target, key))
}
func (s *SceneService) SetSceneProp(name string, value any) error {
	return wrap(s.d.Session.SetSceneProp(name, value))
}
func (s *SceneService) Add(parent uint64, kind string) (uint64, error) {
	return wrap1(s.d.Session.Add(parent, kind))
}
func (s *SceneService) AddComponent(id uint64, kind string) (string, error) {
	return wrap1(s.d.Session.AddComponent(id, kind))
}
func (s *SceneService) RemoveComponent(id uint64, index int) error {
	return wrap(s.d.Session.RemoveComponent(id, index))
}
func (s *SceneService) MoveComponent(id uint64, from, to int) error {
	return wrap(s.d.Session.MoveComponent(id, from, to))
}
func (s *SceneService) ResetComponent(id uint64, index int) error {
	return wrap(s.d.Session.ResetComponent(id, index))
}
func (s *SceneService) Remove(id uint64) error { return wrap(s.d.Session.Remove(id)) }
func (s *SceneService) Reparent(id, parent uint64, index int) error {
	return wrap(s.d.Session.Reparent(id, parent, index))
}
func (s *SceneService) Duplicate(id uint64) (uint64, error) { return wrap1(s.d.Session.Duplicate(id)) }
func (s *SceneService) Unpack(id uint64) error              { return wrap(s.d.Session.Unpack(id)) }

// OpenPrefab edits a prefab on its own; ClosePrefab goes back to the scene.
func (s *SceneService) OpenPrefab(ref string) error { return wrap(s.d.Session.OpenPrefab(ref)) }
func (s *SceneService) ClosePrefab() error          { return wrap(s.d.Session.ClosePrefab()) }

// Apply writes instance overrides into the prefab; name "" applies all.
func (s *SceneService) Apply(id uint64, name string) error {
	return wrap(s.d.Session.Apply(s.d.Ctx.Get(), id, name))
}
func (s *SceneService) Revert(id uint64, name string) error {
	return wrap(s.d.Session.Revert(id, name))
}
func (s *SceneService) SavePrefab(id uint64, rel string) error {
	return wrap(s.d.Session.SavePrefab(s.d.Ctx.Get(), id, rel))
}

type ViewportService struct{ d Deps }

func (v *ViewportService) Resize(cols, rows int)      { v.d.Session.Resize(cols, rows) }
func (v *ViewportService) Snapshot() *session.Frame   { return v.d.Session.Snapshot() }
func (v *ViewportService) Pan(dcol, drow float64)     { v.d.Session.Pan(dcol, drow) }
func (v *ViewportService) Orbit(dyaw, dpitch float64) { v.d.Session.Orbit(dyaw, dpitch) }
func (v *ViewportService) Zoom(k float64)             { v.d.Session.Zoom(k) }
func (v *ViewportService) Focus(id uint64) error      { return wrap(v.d.Session.Focus(id)) }
func (v *ViewportService) Pick(col, row int) uint64   { return v.d.Session.Pick(col, row) }
func (v *ViewportService) Play() error                { return wrap(v.d.Session.Play()) }
func (v *ViewportService) Pause()                     { v.d.Session.Pause() }
func (v *ViewportService) Stop()                      { v.d.Session.Stop() }
func (v *ViewportService) Key(name string) error      { return wrap(v.d.Session.Key(name)) }
func (v *ViewportService) KeyUp(name string) error    { return wrap(v.d.Session.KeyUp(name)) }

// Paint adds a brush stroke on the selected object at a viewport cell.
// Particles drives a particle system from the particle editor.
func (v *ViewportService) Particles(id uint64, key, action, arg string) (int, error) {
	return wrap1(v.d.Session.Particles(id, key, action, arg))
}

func (v *ViewportService) ParticlePresets() []string { return v.d.Session.ParticlePresets() }

func (v *ViewportService) Paint(id uint64, col, row float64, color string, radius float64, erase bool) error {
	return wrap(v.d.Session.PaintAt(id, col, row, color, radius, erase))
}

// AudioEnded reports a play mode sound the web view finished playing.
func (v *ViewportService) AudioEnded(voice uint64) { v.d.Session.AudioEnded(voice) }
func (v *ViewportService) SetLive(on bool)         { v.d.Session.SetLive(on) }
func (v *ViewportService) SetTime(t float64) error { return wrap(v.d.Session.SetTime(t)) }
func (v *ViewportService) DragCollider(id uint64, key, handle string, fc, fr, tc, tr float64) error {
	return wrap(v.d.Session.DragCollider(id, key, handle, fc, fr, tc, tr))
}

func (v *ViewportService) SetGizmos(on bool) { v.d.Session.SetGizmos(on) }

func (v *ViewportService) Turn(id uint64, axis string, fc, fr, tc, tr float64) error {
	return wrap(v.d.Session.Turn(id, axis, fc, fr, tc, tr))
}

func (v *ViewportService) Stretch(id uint64, axis string, fc, fr, tc, tr float64) error {
	return wrap(v.d.Session.Stretch(id, axis, fc, fr, tc, tr))
}

func (v *ViewportService) Drag(id uint64, axis string, fc, fr, tc, tr float64) error {
	return wrap(v.d.Session.Drag(id, axis, fc, fr, tc, tr))
}
func (v *ViewportService) Drop(kind string, col, row float64) (uint64, error) {
	return wrap1(v.d.Session.Drop(kind, col, row))
}
func (v *ViewportService) SetView(mode string) { v.d.Session.SetView(mode) }
func (v *ViewportService) AlignGame()          { v.d.Session.AlignGame() }
func (v *ViewportService) Step()               { v.d.Session.Step() }

type PipelineService struct{ d Deps }

func (p *PipelineService) Passes() []pipeline.Info { return p.d.Session.Passes() }
func (p *PipelineService) SetEnabled(name string, on bool) error {
	return wrap(p.d.Session.SetPassEnabled(name, on))
}
func (p *PipelineService) Move(name string, index int) error {
	return wrap(p.d.Session.MovePass(name, index))
}
func (p *PipelineService) Props(name string) ([]prop.Prop, error) {
	return wrap1(p.d.Session.PassProps(name))
}
func (p *PipelineService) SetProp(name, key string, value any) error {
	return wrap(p.d.Session.SetPassProp(name, key, value))
}

type AssetService struct{ d Deps }

func (a *AssetService) List() ([]asset.Entry, error) { return wrap1(a.d.Session.Assets(a.d.Ctx.Get())) }
func (a *AssetService) Preview(name string) (session.Preview, error) {
	return wrap1(a.d.Session.ArtPreview(name))
}
func (a *AssetService) AddSprite(name string) (uint64, error) {
	return wrap1(a.d.Session.AddSprite(name))
}
func (a *AssetService) Reload() error { return wrap(a.d.Session.Reload(a.d.Ctx.Get())) }

// Reveal shows a project file or folder in the system's file manager.
func (a *AssetService) Reveal(rel string) error {
	p, err := a.d.Session.Path(rel)
	if err != nil {
		return wrap(err)
	}
	return wrap(a.d.Reveal(a.d.Ctx.Get(), p))
}

func (a *AssetService) CreateMaterial(name string) (string, error) {
	return wrap1(a.d.Session.CreateMaterial(a.d.Ctx.Get(), name))
}
func (a *AssetService) Props(ref string) ([]prop.Prop, error) {
	return wrap1(a.d.Session.AssetProps(ref))
}
func (a *AssetService) SetProp(ref, name string, value any) error {
	return wrap(a.d.Session.SetAssetProp(a.d.Ctx.Get(), ref, name, value))
}
func (a *AssetService) Thumb(ref string, size int) (string, error) {
	return wrap1(a.d.Session.Thumb(ref, size))
}

func (a *AssetService) ApplyMaterial(id uint64, ref string) error {
	return wrap(a.d.Session.ApplyMaterial(id, ref))
}
func (a *AssetService) MaterialAt(ref string, col, row int) (uint64, error) {
	return wrap1(a.d.Session.MaterialAt(ref, col, row))
}

func (a *AssetService) Folders() ([]string, error) { return wrap1(a.d.Session.Folders(a.d.Ctx.Get())) }
func (a *AssetService) CreateFolder(rel string) (string, error) {
	return wrap1(a.d.Session.CreateFolder(a.d.Ctx.Get(), rel))
}

// Move puts an asset or folder into folder, "" being the project folder.
func (a *AssetService) Move(from, folder string) (string, error) {
	return wrap1(a.d.Session.Move(a.d.Ctx.Get(), from, folder))
}
func (a *AssetService) RenameFolder(from, name string) (string, error) {
	return wrap1(a.d.Session.RenameFolder(a.d.Ctx.Get(), from, name))
}
func (a *AssetService) Delete(rel string) error { return wrap(a.d.Session.Delete(a.d.Ctx.Get(), rel)) }

func (a *AssetService) CreateController(name string) (string, error) {
	return wrap1(a.d.Session.CreateController(a.d.Ctx.Get(), name))
}
func (a *AssetService) Controller(ref string) (*animator.Controller, error) {
	return wrap1(a.d.Session.Controller(ref))
}
func (a *AssetService) SetController(ref string, c *animator.Controller) error {
	return wrap(a.d.Session.SetController(a.d.Ctx.Get(), ref, c))
}

// ConvertToSprite turns a texture into glyph art cols cells wide.
func (a *AssetService) ConvertToSprite(ref string, cols int) (string, error) {
	return wrap1(a.d.Session.ConvertToSprite(a.d.Ctx.Get(), ref, cols))
}

// Import copies files dropped from the OS into the project; folder "" picks
// one by file type.
func (a *AssetService) Import(paths []string, folder string) ([]string, error) {
	return wrap1(a.d.Session.Import(a.d.Ctx.Get(), paths, folder))
}
func (a *AssetService) Rename(from, to string) (string, error) {
	return wrap1(a.d.Session.RenameAsset(a.d.Ctx.Get(), from, to))
}
func (a *AssetService) NewScript(name string) (string, error) {
	return wrap1(a.d.Session.NewScript(a.d.Ctx.Get(), name))
}

// OpenExternal opens a project file in the user's editor; line is 0 when unknown.
func (a *AssetService) OpenExternal(rel string, line int) error {
	abs, err := a.d.Session.AbsPath(rel)
	if err != nil {
		return wrap(err)
	}
	return wrap(a.d.Edit(a.d.Ctx.Get(), abs, line))
}

type ConsoleService struct{ d Deps }

func (c *ConsoleService) Logs() []session.LogEntry { return c.d.Session.Logs() }
func (c *ConsoleService) Clear()                   { c.d.Session.ClearLogs() }

type AnimService struct{ d Deps }

func (a *AnimService) Eases() []string                         { return anim.Eases() }
func (a *AnimService) Clips(id uint64) ([]session.Clip, error) { return wrap1(a.d.Session.Clips(id)) }
func (a *AnimService) NewClip(id uint64, name string) error {
	return wrap(a.d.Session.NewClip(id, name))
}
func (a *AnimService) PickClip(index int) { a.d.Session.PickClip(index) }
func (a *AnimService) RenameClip(index int, name string) error {
	return wrap(a.d.Session.RenameClip(index, name))
}
func (a *AnimService) AnimatorState(id uint64) string { return a.d.Session.AnimatorState(id) }

func (a *AnimService) AddKey(id uint64, name string, t float64) error {
	return wrap(a.d.Session.AddKey(id, name, t))
}
func (a *AnimService) RemoveKey(index int, name string, t float64) error {
	return wrap(a.d.Session.RemoveKey(index, name, t))
}
func (a *AnimService) SetClip(index int, mode string, speed float64) error {
	return wrap(a.d.Session.SetClip(index, anim.Mode(mode), speed))
}
func (a *AnimService) RemoveClip(index int) error { return wrap(a.d.Session.RemoveClip(index)) }
func (a *AnimService) MoveKey(index int, name string, from, to float64) error {
	return wrap(a.d.Session.MoveKey(index, name, from, to))
}
func (a *AnimService) SetEase(index int, name string, t float64, ease string) error {
	return wrap(a.d.Session.SetEase(index, name, t, anim.Ease(ease)))
}

type BuildService struct{ d Deps }

func (b *BuildService) Targets() []string { return b.d.Session.Targets() }

// Build writes a game folder (player, packed data, checksums) for target,
// or one executable when single is set.
func (b *BuildService) CanBrand() bool { return b.d.Session.CanBrand() }
func (b *BuildService) Build(target string, o session.BuildOptions) (session.BuildResult, error) {
	return wrap1(b.d.Session.Build(b.d.Ctx.Get(), target, o))
}
func (b *BuildService) Reveal(path string) error { return wrap(b.d.Reveal(b.d.Ctx.Get(), path)) }
