# ASCDII Engine

2.5D Braille render engine for the terminal, an example game and a desktop editor.

## Layout

One Go module.

- `pkg/ascdii` public facade: type aliases and constructors for engine users.
  Nothing else lives under `pkg/`.
- `internal/engine/...` the engine, one package per concern, grouped by module:
  `core` (color, geom, prop, input, ...), `render` (raster, texture, draw,
  pipeline, ...), `world` (scene: GameObjects, components, references;
  components; anim; fx), `assets` (asset, mesh, material), `script/vm` (Lua),
  `platform` (backend contract, terminal, headless, pixels, window), `runtime` (app, project,
  pack, bundle, ...).
- `internal/dragonfall/...` example game content built on the engine.
- `internal/editor/...` editor backend (session, services, Wails adapters).
- `ascdii` without arguments (and `ascdii run`) starts the editor
  (`internal/editor/launch`); with arguments it is the CLI. An editor started
  directly restarts itself through `ascdii run`, except in `wails dev`.
  `ascdii help` prints the commands; `ascdii play` runs a project.
- `cmd/ascdii` CLI, `cmd/player`
  ascdii-player that games run on (`internal/player`), `cmd/brand` icons and
  Windows resources, `cmd/hello` minimal example, `cmd/snapshot`
  PNG renders for review, `cmd/editor` Wails app (built only with Wails tags).
- `internal/dragonfall/template` is generated: `make template` after editing
  `internal/dragonfall/scene.go`.
- `frontend/` Vue 3 + TypeScript + Tailwind editor UI.

Dependency direction: `editor -> dragonfall -> engine`, `editor -> engine`. The
engine never imports either.

## Rules

- Never write em-dashes, in code, comments or docs.
- Comments explain why, never what. Do not comment obvious code, do not split
  logic or struct fields with comment lines, and do not write long comment
  sections. Short trailing notes on a field are fine.
- Build bottom up: the model a step needs, then the service that uses it, then
  the next model. Not all models first, not a service before its model.
- Dependencies are injected. More than three go in a `Deps` struct. No package
  level state, no `init()` that connects to anything.
- Errors wrap with `%w`, start lowercase, are handled or returned, not both.
  There is no HTTP layer; module sentinel errors are mapped to codes at the
  editor boundary (`internal/editor/service`).
- `ctx context.Context` is the first parameter of anything that does I/O.
- Tests use the standard library, sit next to their subject, run under `-race`,
  are named `TestUnit_Behaviour` and use Arrange / Act / Assert comments.
- Add a dependency in the same commit that uses it. Standard library first.
- Embedded files (`frontend/dist`, templates) and the linker-set game data
  key (`bundle.pakSecret`) are the only package level values; they are
  read-only build artefacts.
- The game data key comes from `.env` (`ASCDII_PAK_SECRET`, git ignored,
  written by the first `make`); never put a real key in the source.

## Assets

Packages (`runtime/pack`) are installed into `packages/<name>/` and are read
only: the session refuses writes there (`ErrPackageLocked`). Go components ship
as code modules (`project.Module`); a project loads only the modules its
manifest lists.

Every asset has a `<file>.meta` with a UUIDv7 (`internal/engine/core/guid`) and
optional import settings. Scenes reference assets by that id;
`asset.Library.Resolve` turns ids (or legacy paths) into files. Objects inside a
scene or prefab have a random file id (`fid`); object and component references
use it (`scene.RefOf`, `scene.Lookup`). Prefab instances store overrides, not
copies. Generated content uses `scene.CountingFIDs` so it stays deterministic.

## Units

Content is authored in world units only. Dots (2x4 per cell) and cells exist
inside `internal/engine/render/raster` and the resolver. Sprite art uses art pixels (y down, origin
at the feet) scaled by `Sprite.Scale` world units per pixel.

## Commands

- `make test` runs `go vet` and `go test -race ./...`.
- Docs: `python3 docs/nav.py` rewrites page navigation; `go run ./cmd/docref
  docs/usage/components.html` (and `docs/usage/dragonfall.html dragonfall`)
  regenerates the component reference. Every component property needs a
  description in `cmd/docref/notes.go`.
- `docs/internals/*.md` is the engine developer reference (packages, types,
  functions, flows, decisions, debugging notes); `python3
  docs/internals/build.py` regenerates its `.html` pages and search index.
- The engine version is set only in `internal/engine/runtime/version/version.go`.
  `make version` (part of `make all`) copies it into `cmd/editor/wails.json`, the
  Windows resources, the Dragonfall template and the docs; the frontend reads it
  at build time. Never write the version anywhere else; in docs/usage pages wrap
  it in `<span class="version">`, which `docs/nav.py` fills in.
- `make hello`, `make dragonfall` run the terminal examples.
- The window display (`platform/window`, Ebitengine) is linked only with the
  `window` build tag, into `ascdii-player-window`; never import it from code
  the terminal programs or the editor link.
- `make all` builds `bin/linux` and `bin/windows`, each a complete install
  of that platform only (ascdii, ascdii-editor, ascdii-player,
  ascdii-player-window) that runs where it is copied. Only these four are shipped; the other commands are
  run with `go run`. Parts: `make cli`, `make editor-linux`, `make editor-windows`;
  `make editor-dev` for hot reload.
