# Web UI

Hermeneia's first web UI lives in `apps/web`. It is a SvelteKit app for local content operations and uses the Go HTTP API as its source of truth.

## Current Scope

The MVP slice supports:

- listing content runs,
- inspecting a selected run,
- reviewing brief versions,
- viewing revision history,
- previewing generated image/video artifacts grouped by kind,
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
form, but run creation still calls the existing run endpoint. Workflow execution
by `workflow_id` remains a later backend slice.

Run detail shows a derived timeline from existing run data:

- research artifacts,
- brief versions,
- revision events,
- render artifacts,
- scheduled publishing records.

## Validation

The initial view-model helpers can be tested without installing the full SvelteKit toolchain:

```bash
cd apps/web
npm test
```
