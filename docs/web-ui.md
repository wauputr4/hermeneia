# Web UI

Hermeneia's first web UI lives in `apps/web`. It is a SvelteKit app for local content operations and uses the Go HTTP API as its source of truth.

## Current Scope

The MVP slice supports:

- listing content runs,
- inspecting a selected run,
- reviewing brief versions,
- viewing revision history,
- filtering generated artifacts by kind,
- previewing image/video artifacts while keeping text/json artifacts compact,
- opening or downloading individual artifacts through the local API file endpoint,
- running the read-only artifact integrity audit for the selected run,
- creating a run with a workflow-aware and template-aware form,
- reviewing selected workflow metadata and ordered steps before run creation,
- viewing a derived step timeline for the selected run,
- saving deterministic revision instructions,
- triggering render/export jobs.

The UI does not own business logic. It calls the API documented in [HTTP API](api.md), and the API calls the same workflow service used by the CLI.

## Local Development

Start the Go API:

```bash
go run ./cmd/hermeneia serve --addr 127.0.0.1:19318
```

Start the SvelteKit app:

```bash
cd apps/web
npm install
npm run dev
```

Configuration:

```text
PUBLIC_HERMENEIA_API_BASE=http://127.0.0.1:19318
```

When unset, the app defaults to `http://127.0.0.1:19318`.

## Workflow Selection

The create-run form reads `GET /v1/workflows` and lets users select a preset.
Selecting a workflow updates the content type and default template shown in the
form. When a workflow is selected, run creation sends `workflow_id` to
`POST /v1/runs`, so executable presets can create a run through their ordered
service steps. Presets with a `render` step return generated artifact metadata
and the run detail refresh shows those artifacts immediately after creation.

Choosing `Manual run` omits `workflow_id` and keeps the previous direct
`content_type` plus `template_id` creation path for unrendered draft runs.

Run detail shows a derived timeline from existing run data:

- research artifacts,
- brief versions,
- revision events,
- render artifacts,
- scheduled publishing records.

## Artifact Browser

The artifact section reads artifact metadata from the selected run detail
payload and uses the existing local-only file endpoint for open/download links:

```text
GET /v1/runs/{run_id}/artifacts/{artifact_id}/file
```

Image and video artifacts render inline previews. Text and JSON artifacts keep a
compact metadata row with filename, path, timestamp, checksum status, and direct
links.

The run detail view also calls the read-only artifact audit endpoint on demand:

```text
GET /v1/runs/{run_id}/artifact-audit
```

Healthy audits show an empty-issue state. Drift responses that return
`409 Conflict` still render their structured issue payload in the UI, including
the issue kind, artifact ID when available, path, and message. The Web UI does
not repair files, delete untracked output, or change SQLite artifact rows.

## Validation

The initial view-model helpers can be tested without installing the full SvelteKit toolchain:

```bash
cd apps/web
npm test
```
