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
- Use `hermeneia audit <run-id>` to check a run for missing artifact files,
  checksum mismatches, unsafe stored paths, and untracked files under
  `runs/{run-id}/output/`.

## Artifact integrity audit

`hermeneia audit <run-id>` is read-only. It exits successfully when every
artifact row points to an existing file inside the run directory, stored
checksums still match, and every file under `output/` is tracked in SQLite.

If the audit reports drift:

- `missing_file`: preserve the database row and rerender the run if the output
  file cannot be restored.
- `checksum_mismatch`: inspect the file for manual edits, then rerender or
  intentionally regenerate metadata in a later repair workflow.
- `unsafe_path`: treat the row as invalid because the path points outside the
  run folder.
- `untracked_file`: remove the stray output file or add a deterministic repair
  path before relying on it operationally.

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

The local HTTP API is exposed through `hermeneia serve --addr 127.0.0.1:19318`.

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
go run ./cmd/hermeneia serve --addr 127.0.0.1:19318
```

Notes:

- The frontend defaults to `http://127.0.0.1:19318`; set `PUBLIC_HERMENEIA_API_BASE` if the API runs elsewhere.
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

## 2026-05-10 — Template manifest loader

Built-in templates are loaded from local `templates/**/template.json` manifests.

Notes:

- Template IDs must map directly to manifest paths, for example `carousel/ai-news-clean` must live at `templates/carousel/ai-news-clean/template.json`.
- Required manifest fields are documented in `docs/templates.md`.
- The loader rejects missing fields, duplicate IDs, unsupported content types, invalid `input_schema` JSON, and ID/path mismatches before content runs are created.
- `hermeneia create` and `hermeneia research` use the catalog to select default built-in templates and reject templates whose `content_type` does not match the requested run type.
- Built-in template discovery should find the nearest ancestor `templates/` directory that actually contains at least one `template.json`; it must not require `go.mod`, so copied binary-plus-template deployments can still run.
- Keep `workflow.NewService` preloaded with the built-in catalog when possible so repeated create/research calls do not rescan the template tree on every request.
- Use `hermeneia templates` or `GET /v1/templates` to inspect the active template catalog before creating runs with explicit template IDs.
- Template API responses must omit local `Path` details from manifests; expose IDs and metadata only.

## 2026-05-11 — Web UI template catalog review

The Web UI template picker should treat API catalog data as external input:

- Sort compatible templates through the same display label helper used by the UI so missing or blank names fall back to the template ID.
- When loading templates, validate the current `template_id` against templates compatible with the selected `content_type`, not only against the full catalog.
- Render optional manifest arrays such as `output_kinds` defensively so malformed local template manifests do not crash the create-run form.

## 2026-05-11 — Custom template directories

Set `HERMENEIA_TEMPLATE_PATH` to one or more template roots separated by the
operating system path-list separator (`:` on Linux/macOS) to add local templates
alongside built-ins.

Notes:

- Custom roots that contain no `template.json` files are ignored; catalog loading only fails with a missing-manifest error when no configured root contributes any templates.
- Template IDs still have to map to their manifest path under the root, such as `carousel/local-clean` -> `carousel/local-clean/template.json`.
- Duplicate template IDs across built-in and custom roots fail instead of overriding an existing template.
- `preview_asset` and `assets` entries must stay inside the template directory; absolute paths and `../` traversal are rejected.

If a custom template is not listed:

- Confirm the template ID matches the path below the configured root, such as
  `carousel/local-clean` at `carousel/local-clean/template.json`.
- Check for duplicate IDs with built-ins or earlier custom roots.
- Use only supported content types: `carousel` or `short_video`.
- Verify each declared `preview_asset` or `assets` entry exists inside the
  template directory.
- Run `HERMENEIA_TEMPLATE_PATH=/path/to/root hermeneia templates` before
  creating a run so manifest errors are isolated from workflow errors.

## 2026-05-11 — Template input schema validation

Hermeneia validates the rendered structured input against a manifest's
`input_schema` before run creation and again before `render` writes
`content.json` or output artifacts.

Notes:

- The supported schema subset is intentionally small: `type`, `required`, `properties`, `items`, `minItems`, `maxItems`, `const`, `enum`, and numeric `minimum`.
- Use `maxItems` on carousel `slides` or video `scenes` to fail fast before partial render output is written.
- Validation errors are returned through the CLI and HTTP API as normal invalid-input errors.

## 2026-05-11 — Workflow preset contract

Workflow presets live under `workflows/` as JSON contracts that map named flows
to existing service steps. They intentionally do not execute arbitrary scripts.

## 2026-05-17 — Workflow preset execution limits

`hermeneia create --workflow <id>` and `POST /v1/runs` with `workflow_id`
support only ordered create-run sequences: `create_brief`, `create_brief` then
`render`, `research_plan` then `create_brief`, and `research_plan` then
`create_brief` then `render`.

If execution fails with an unsupported step order error, inspect the preset's
`steps` array. `revise_brief`, `schedule_record`, and reordered research steps
are valid catalog metadata, but they are not executed during create-run flows
yet. Use the normal revise or schedule commands after creating the run, or split
the preset into a supported create-run sequence.

Notes:

- Presets must use supported content types: `carousel` or `short_video`.
- Supported step types are `create_brief`, `research_plan`, `revise_brief`, `render`, and `schedule_record`.
- `default_template_id` must reference an installed template whose manifest content type matches the preset content type.
- Duplicate preset IDs fail validation instead of overriding earlier presets.
- Required preset fields are validated in a fixed order for deterministic errors. `required_inputs` must be non-empty so upcoming CLI/API/UI catalog consumers know which operator inputs are needed before execution.
- Use `hermeneia workflows`, `GET /v1/workflows`, or `GET /v1/workflows/{workflow_id}` to inspect built-in preset metadata. Preset execution is intentionally separate from this read-only catalog slice.
- Built-in workflow discovery stops scanning once it finds a preset JSON file, and the workflow service caches lazy-loaded preset catalogs on the service instance. If repeated CLI/API workflow catalog calls look slow, confirm callers keep a stable service instance instead of constructing one per request.

If a workflow preset fails validation:

- Confirm `required_inputs` is present and non-empty.
- Use only supported step types: `create_brief`, `research_plan`,
  `revise_brief`, `render`, and `schedule_record`.
- Ensure `default_template_id` exists in the active template catalog.
- Match the preset `content_type` to the default template's manifest
  `content_type`.
- Rename duplicate preset IDs instead of relying on override order.

## 2026-05-16 — Workflow preset execution

`hermeneia create --workflow <id>` and `POST /v1/runs` with `workflow_id` can
create normal runs from workflow presets.

Notes:

- The preset content type and default template are used for run creation.
- `topic` is required for built-in create flows.
- Presets with `research_plan` require at least one source URL through
  `--source` or API `sources`.
- Presets with `render` call the existing renderer and return standard artifact
  metadata.
- Unsupported required input names fail with a validation error instead of being
  ignored.
- Presets still cannot run shell commands, plugins, arbitrary scripts, or
  external publishing connectors.

## 2026-05-11 — Web UI workflow selector and timeline

The Web UI workflow selector uses the read-only workflow catalog. Selecting a
preset should update the create form's content type and default template, but it
must not send `workflow_id` to `POST /v1/runs` until backend workflow execution
is implemented.

The run detail step timeline is derived from existing detail response data. If a
step looks wrong, inspect the run's brief versions, `research_json` artifact,
render artifacts, revision events, and `scheduled_posts` payload before changing
the UI state model.

Treat those API arrays as external data: sort briefs, revisions, render
artifacts, and scheduled posts before selecting display timestamps. The brief
step label and timestamp should both use the latest brief version; revision and
render timestamps should use the latest `created_at`; schedule timestamps should
use the latest `scheduled_at`.

## 2026-05-12 — Web UI artifact browser

The artifact browser filters run artifacts by metadata kind and links each file
through `GET /v1/runs/{run_id}/artifacts/{artifact_id}/file`.

Notes:

- Image and video artifacts should preview inline, but text/json artifacts
  should remain compact metadata rows.
- If an artifact link fails, verify the artifact still belongs to the selected
  run and that the stored path stays inside the run directory. The API file
  handler enforces this local boundary.
- Browser download behavior can vary for cross-origin development servers; the
  direct open link uses the same safe endpoint.
- Artifact rows should tolerate missing stored paths and fall back to the
  artifact ID for display labels instead of crashing the run detail page.

## 2026-05-10 — API-driven Web UI template gallery

The Web UI create-run form loads templates from `GET /v1/templates` instead of
keeping a hardcoded frontend list.

Notes:

- Start the Go API before opening the SvelteKit app; otherwise the template
  selector shows a user-facing catalog load error.
- Template filtering uses manifest `content_type` values such as `carousel` and
  `short_video`.
- If the template catalog is empty or has no compatible template for the
  selected content type, the create button stays disabled until the API returns
  a usable template ID.
