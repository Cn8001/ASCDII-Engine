# ASCDII Engine 0.0.1

A 2.5D render engine that draws into Unicode Braille. Every terminal cell holds
2x4 depth tested, lit subpixels that are reduced to one foreground and one
background color. Games run in any truecolor terminal; the editor is a Wails
desktop app.

## Quick start

```bash
make test                       # go vet + go test -race
make hello                      # smallest program, see cmd/hello
make dragonfall                 # the sample game in the terminal
make cli players editor-linux   # bin/: the ascdii launcher, players, the editor
bin/ascdii                      # launcher: recent projects open in the editor
make editor-dev                 # editor with hot reload, without the launcher
```

```go
e := ascdii.New(ascdii.Options{})
s := e.Scene()
s.Add("ground", ascdii.Ground(40, 40, ascdii.Hex("#46623e")))
s.Add("crate", ascdii.Box(ascdii.V3(2, 2, 2), ascdii.Hex("#b08050"))).Blocking()
s.Add("hero", ascdii.Blob(.8, ascdii.Hex("#e8b04b"))).At(0, 5, 0).With(ascdii.Player())
s.Animate("crate", ascdii.Tween("pos.z", 0.0, 1.5, 1.2, ascdii.InOut).Bouncing())
err := e.Run(ctx, os.Stdin, os.Stdout)
```

## Projects

A game is a folder you can keep in git:

```
MyGame/
  MyGame.ascdiiproj      manifest: name, version, engine, start scene, modules
  **/*.scene.json        GameObject trees, environment, animation clips
  **/*.prefab.json       reusable GameObject trees, instanced by reference
  **/*.obj, *.png        models with their .mtl files, textures
  **/*.material.json, *.animator.json
  **/*.sprite.json, *.xp sprites: glyph art as palette JSON or REXPaint
  **/*.lua               Lua scripts with Inspector fields
  **/*.wav, *.mp3        sounds
  packages/<name>/       installed packages, read only
  **/*.meta              UUIDv7 per asset, plus import settings
  build/                 output, ignored by the generated .gitignore
```

Only the manifest has a fixed place; folders are the project's own.

CLI, also the game player:

```bash
ascdii new MyGame -template dragonfall
ascdii run MyGame
ascdii preview MyGame -size 160x50 -time 2
ascdii build MyGame                          # folder: player, data/game.pak, SHA256SUMS
ascdii build MyGame -single                  # one executable instead
ascdii pack install MyGame dragonfall        # the sample as a package
ascdii pack export MyGame/props -name props  # share a folder as a .ascdiipack
ascdii version
```

A build is the `ascdii` binary as the player next to `data/game.pak`, the
project compiled into one encrypted file, plus checksums. The build is the
release.

## Editor

`make editor-linux` and `make editor-windows` write `cmd/editor/build/bin/`
and copy the editor next to `bin/ascdii`. The editor is started from the
`ascdii` launcher, which lists recent projects and opens the chosen one; started
directly, it opens the launcher instead. `make editor-dev` skips the launcher.
The editor has a GameObject menu, a Hierarchy (drag to reparent, right click to
create, duplicate, save as prefab), a live 2.5D viewport with move, rotate and
scale tools, an Inspector with component cards and reference slots, a Project
window with free folders and file import, the pipeline, timeline, animator,
particle editor, console and builds.

Packages are assets with their .meta files and a pack.json,
shared as one `.ascdiipack`, installed into `packages/` read only and removed
as a whole from the Package Manager. New projects start empty; the Dragonfall
sample is a package, and its Go components are a code module that only
projects using it load.

GameObjects carry any number of components. Lua scripts are components too:
they declare `fields` the Inspector edits and reach objects, components, the
camera, the environment and the render pipeline (see `docs/usage/lua-api.html`).

Keys: Space play, Esc stop, Ctrl+S save, Ctrl+Z / Ctrl+Y undo and redo,
Ctrl+D duplicate, Del delete, F focus, Ctrl+B build.

## Render model

- One unit: world units. Dots and cells exist only inside the rasterizer.
- Camera with pitch and yaw; depth comes from the projection, never by hand.
- Deferred G-buffer per dot: albedo, normal, depth, world position, tag.
- Passes: clear, scene, shadows, lighting, water, fog, outline, grade,
  braille, cells. Add your own with `ascdii.Func(name, stage, fn)`.
- Collision reads the same bounds that drawing uses.

## Documentation

Open `docs/usage/index.html` in a browser: an offline tutorial covering install,
a first game, the editor, scenes and prefabs, animation, Lua scripting,
rendering, building and the Go API.

## Layout

`pkg/ascdii` is the only public package. The engine lives in
`internal/engine/{core,render,world,assets,script,platform,runtime}`, the
sample content in `internal/dragonfall`, the editor backend in
`internal/editor`, the UI in `frontend/`, programs in `cmd/` (the terminal
menu is `cmd/ascdii/menu`). `docs/usage/architecture.html` explains the modules
and the display backend contract. See `CLAUDE.md` for rules.
