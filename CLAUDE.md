# ASCDII Engine

2.5D Braille render engine for the terminal, an example game and a desktop editor.

## Layout

One Go module.

- `pkg/ascdii` public facade: type aliases and constructors for engine users.
  Nothing else lives under `pkg/`.
- `internal/engine/...` the engine, one package per concern (color, geom, raster,
  scene, pipeline, ...).
- `internal/dragonfall/...` example game content built on the engine.
- `internal/editor/...` editor backend (session, services, Wails adapters).
- `internal/launcher` the terminal main menu `ascdii` shows without arguments.
- `cmd/ascdii` CLI and game player, `cmd/hello` minimal example, `cmd/snapshot`
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
- Embedded files (`frontend/dist`, players, templates) are the only package
  level values; they are read-only build artefacts.

## Assets

Every asset has a `<file>.meta` with a UUIDv7 (`internal/engine/guid`). Scenes
reference prefabs, scripts and art by that id; `asset.Library.Resolve` turns ids
(or legacy paths) into files. Prefab instances store overrides, not copies.

## Units

Content is authored in world units only. Dots (2x4 per cell) and cells exist
inside `internal/engine/raster` and the resolver. Sprite art uses art pixels (y down, origin
at the feet) scaled by `Sprite.Scale` world units per pixel.

## Commands

- `make test` runs `go vet` and `go test -race ./...`.
- `make hello`, `make dragonfall` run the terminal examples.
- `make players` before any editor build; `make editor-dev`, `make editor-linux`,
  `make editor-windows`.
