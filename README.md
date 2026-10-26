# ASCDII Engine 0.0.1

A 2.5D game engine that draws into Unicode Braille. Every terminal cell holds
2x4 depth tested, lit subpixels that are reduced to one foreground and one
background color. Games run in any truecolor terminal; the editor is a Wails
desktop app.

## Quick start

```bash
make test                       # go vet + go test -race
make hello                      # smallest program, see cmd/hello
make dragonfall                 # the sample game in the terminal
make all                        # bin/linux and bin/windows: ascdii, editor, player
bin/linux/ascdii                # opens the editor at its start screen
make editor-dev                 # editor with hot reload
```

```go
e := ascdii.New(ascdii.Options{})
s := e.Scene()
s.Add("camera", ascdii.CameraComponent(34))
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

## CLI

`ascdii` without arguments opens the editor at its start screen, so a double
click on `ascdii.exe` starts it like any desktop program. With arguments it is
the command line tool:

| Command | Function |
|---|---|
| `ascdii` | The editor, at its start screen |
| `ascdii run [project]` | The editor, with a project open, given as its folder or `.ascdiiproj` |
| `ascdii new <dir> [-name N] [-template dragonfall]` | Create a project |
| `ascdii play <dir> [--blocks]` | Play a project in the terminal; `--blocks` uses half blocks instead of Braille |
| `ascdii preview <dir> [-size 160x50] [-time 2] [-hour 22] [-plain] [-blocks]` | Print one frame after simulating `-time` seconds |
| `ascdii build <dir> [-out path] [-player path] [-single]` | Build a game folder: player, `data/game.pak`, `SHA256SUMS`; `-single` builds one executable |
| `ascdii pack list <dir>` | List installed and built in packages |
| `ascdii pack install <dir> <file.ascdiipack \| dragonfall>` | Install a package |
| `ascdii pack remove <dir> <name>` | Remove a package |
| `ascdii pack export <folder> -name N [-version V] [-description D] [-out file]` | Share a folder as a `.ascdiipack` |
| `ascdii version` | Print the engine version |
| `ascdii help` | Print the list of commands |

```bash
ascdii new MyGame -template dragonfall
ascdii play MyGame
ascdii build MyGame
```

`ascdii-player <dir>` also plays a project folder; a built game is a renamed
`ascdii-player` and starts without arguments.

A build is the `ascdii-player` binary, renamed, next to `data/game.pak`, the
project compiled into one encrypted file, plus checksums. The build is the
release.

## Editor

`make editor-linux` and `make editor-windows` put the editor next to
`bin/linux/ascdii` and `bin/windows/ascdii.exe`, which start it at its start
screen when run without arguments; started directly, the editor restarts
itself through `ascdii run`.
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
`internal/editor`, the UI in `frontend/`, programs in `cmd/` (the editor
is started by `cmd/ascdii`, games run on `cmd/player`). `docs/usage/architecture.html` explains the modules
and the display backend contract. See `CLAUDE.md` for rules.
