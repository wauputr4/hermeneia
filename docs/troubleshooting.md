# Troubleshooting Notes

This file should be updated after meaningful implementation work.

## Early Considerations

### Remotion License

Remotion has special license terms. It is free for individuals, small for-profit organizations up to certain limits, non-profits, and evaluation use cases. Before commercial production use, review Remotion's official license terms.

### Renderer Boundary

Go should not directly own React rendering logic. Use structured JSON as the boundary between Go workflow services and TypeScript renderers.

### Revision History

Avoid overwriting generated content during revisions. Store versions explicitly.

### Social Platform APIs

Publishing integrations should be added later because Meta, YouTube, and TikTok APIs introduce authentication, rate limits, review requirements, and platform-specific media constraints.

## 2026-05-07 — MVP Scope Locked

Added `docs/mvp.md` to define the first usable Hermeneia MVP and explicit non-goals.

Important direction:

- MVP prioritizes CLI + SQLite + structured brief/revision workflow.
- Web UI, full AI research automation, and publishing integrations are intentionally deferred.
- Generated artifacts may live in `runs/`, but SQLite should track metadata and revision history.

## MVP storage/source-of-truth split

- SQLite is the queryable source of truth for run metadata, brief versions, revision events, template selections, and artifact references.
- The `runs/` directory remains the source of truth for exported asset bytes and inspectable JSON/Markdown snapshots.
- If a database row and file artifact disagree, prefer preserving the file artifact and repair/rebuild the SQLite metadata from the run folder where possible.

## SQLite initialization

If `hermeneia init` cannot open the database, check `HERMENEIA_DATABASE_PATH` first. The default path is `data/hermeneia.db`; parent directories must exist before initialization.

For tests or temporary local runs, use an isolated path:

```bash
HERMENEIA_DATABASE_PATH=/tmp/hermeneia.db go run ./cmd/hermeneia init
```

## CLI command tests

The CLI entrypoint should call the shared `run` helper so production execution and tests use the same command initialization path. Unit tests that instantiate `command` directly should provide a `stdout` writer, even when the exercised path is expected to return an error, so future command output cannot panic on a nil writer.

## Brief schema examples

When changing the MVP brief schema, update the Go struct in `internal/brief`, the schema documentation, and the committed example JSON together. Tests load the example file directly, so schema drift should be caught by `go test ./...`.

## 2026-05-07 — SQLite migration review follow-up

PR review feedback on the initial storage layer clarified three reliability rules:

- `storage.Open` should create parent directories for file-backed SQLite database paths before opening the database; `:memory:` remains unchanged for tests.
- Schema migrations should run inside a transaction and record an applied schema version in `schema_migrations` so future migrations have a clear upgrade path.
- Repository queries should avoid redundant SQLite `json(...)` calls when the schema already validates JSON with `CHECK (json_valid(body_json))`.

## 2026-05-08 — CLI skeleton

The first CLI skeleton exposes the MVP command surface in help output while only `hermeneia init` performs real work.

Notes:

- `hermeneia init` uses `HERMENEIA_DATABASE_PATH` or defaults to `data/hermeneia.db`.
- Planned commands (`create`, `list`, `show`, `revise`, `render`) intentionally return clear "not implemented yet" errors until the workflow services are implemented.
- Unknown commands should point users back to `hermeneia help` instead of failing silently.

## 2026-05-09 — CLI MVP workflow and render path

The MVP CLI now supports `create`, `list`, `show`, `revise`, and `render`.

Notes:

- `create` writes `brief.v1.json`, creates the deterministic run folder, and stores run metadata in SQLite.
- `research` writes `research.json`, preserves supplied source URLs, records a `research_json` artifact row, and creates a schema-valid brief draft from the research plan.
- `revise` appends a new `brief.v{n}.json` and records a revision event without overwriting older brief files.
- MVP revision behavior is deterministic: the CLI records the human instruction, appends a visible revision note, and preserves version history. It is not yet an LLM rewrite pipeline.
- `render` writes `content.json`, renderer outputs, artifact rows, and checksums.
- Carousel output is generated as PNG slides under `runs/{run-id}/output/carousel/`.
- Video output writes `output/video/remotion-input.json` and `output/video/ai-news-short.mp4`.
- The temporary local MP4 renderer requires `ffmpeg` on `PATH`; the Remotion scaffold in `packages/renderer-video` uses the same structured JSON contract for the future TypeScript renderer.

If video rendering fails with an `ffmpeg is required` message, install `ffmpeg` locally or render only carousel runs until the Remotion worker is wired into the CLI.
