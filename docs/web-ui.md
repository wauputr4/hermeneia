# Web UI

Hermeneia's first web UI lives in `apps/web`. It is a SvelteKit app for local content operations and uses the Go HTTP API as its source of truth.

## Current Scope

The MVP slice supports:

- listing content runs,
- inspecting a selected run,
- reviewing brief versions,
- viewing revision history,
- viewing artifact paths grouped by kind,
- creating a run with a template-aware form,
- saving deterministic revision instructions,
- triggering render/export jobs.

The UI does not own business logic. It calls the API documented in [HTTP API](api.md), and the API calls the same workflow service used by the CLI.

## Local Development

Start the Go API:

```bash
go run ./cmd/hermeneia serve --addr 127.0.0.1:19317
```

Start the SvelteKit app:

```bash
cd apps/web
npm install
npm run dev
```

Configuration:

```text
PUBLIC_HERMENEIA_API_BASE=http://127.0.0.1:19317
```

When unset, the app defaults to `http://127.0.0.1:19317`.

## Validation

The initial view-model helpers can be tested without installing the full SvelteKit toolchain:

```bash
cd apps/web
npm test
```
