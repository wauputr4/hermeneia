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

## 2026-05-09 — HTTP API MVP slice

The local HTTP API is exposed through `hermeneia serve --addr 127.0.0.1:19317`.

Notes:

- API handlers live under `internal/httpapi`.
- Handlers call the same `workflow.Service` methods as the CLI.
- Route docs live in `docs/api.md`.
- API responses should use rows loaded back from SQLite when they expose database-owned timestamps. Batch-load newly inserted render artifacts so timestamp hydration does not add one read query per generated file.
- The API is intentionally local-first; authentication and multi-user concerns
  are deferred until hosted collaboration becomes part of the product scope.

## 2026-05-09 — OpenAI research planner

`hermeneia research` stays deterministic by default. Use `--planner openai` only when the process has both `OPENAI_API_KEY` and `OPENAI_MODEL` configured.

Notes:

- OpenAI planning calls the Responses API and requests structured JSON for the stored research plan.
- Hermeneia overwrites the returned plan's `sources`, `topic`, `content_type`, and `template_id` with local request values before writing `research.json`, so source URL traceability remains under local control.
- If `--planner openai` fails because configuration is missing, either set the two environment variables or omit the flag to use the deterministic planner.

## 2026-05-09 — Web UI Svelte 5 review notes

The web UI is a Svelte 5 app. New components should use runes state (`$state`,
`$derived`, `$effect`) and Svelte 5 event attributes (`onclick`, `onsubmit`,
`onchange`) to avoid legacy-mode warnings during `npm run build`.

The shared web API request helper should normalize custom headers through the
`Headers` constructor before adding the default JSON content type. Typed JSON
requests intentionally fail on HTTP 204 responses so list/detail callers do not
silently receive `undefined`.

Cache reusable `Intl.DateTimeFormat` instances in view-model helpers and return
a fallback such as `n/a` for invalid timestamps so malformed API data cannot
crash the dashboard.

## 2026-05-09 — SvelteKit web UI slice

The local web UI in `apps/web` expects the Go API to be running first:

```bash
go run ./cmd/hermeneia serve --addr 127.0.0.1:19317
```

Notes:

- The frontend defaults to `http://127.0.0.1:19317`; set `PUBLIC_HERMENEIA_API_BASE` if the API runs elsewhere.
- Browser CORS behavior may matter once the dev server and API use different origins. Keep local API development on loopback and add explicit API CORS handling before exposing it outside local development.
- Run `npm test` inside `apps/web` for view-model helper coverage. Full SvelteKit build validation requires installing frontend dependencies with `npm install`.

## 2026-05-09 — Scheduling foundation

`hermeneia schedule` creates local scheduling metadata only. It does not publish
to Meta, YouTube, TikTok, LinkedIn, or other external platforms yet.

Important guardrails:

- SQLite stores schedule status, platform name, selected artifact id, and validation metadata.
- SQLite must not store OAuth tokens, API keys, refresh tokens, or account credentials.
- Supported MVP platform names are `instagram`, `facebook`, `youtube`, `tiktok`, and `linkedin`.
- Use future RFC3339 timestamps for `--at` and API `scheduled_at` values. Past timestamps are rejected before schedule rows are created.
