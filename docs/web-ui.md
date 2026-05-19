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
- creating local schedule records for rendered artifacts,
- reviewing selected-run local schedule history,
- viewing a read-only scheduled-post agenda across local runs,
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

## Local Scheduling

The run detail operations panel can create a local schedule record after a run
has at least one non-research artifact. The form sends:

```text
POST /v1/runs/{run_id}/schedule
```

Users choose an artifact, a supported platform, and a future browser-local date
and time. The UI converts the selected time to an RFC3339 timestamp for the API,
then refreshes the selected run so the step timeline shows the new scheduled
post. This is local metadata only: the MVP does not store platform credentials
and does not call Meta, YouTube, TikTok, LinkedIn, or other publishing APIs.
The selected run detail area shows the same run-local schedule records in a
compact history list using `selectedDetails.scheduled_posts`, ordered by
`scheduled_at` with schedule ID as a deterministic fallback. Rows show schedule
ID, platform, status, artifact ID or `none`, and scheduled time without making
another API request.

The sidebar agenda reads:

```text
GET /v1/scheduled-posts
```

The UI sends API query parameters for active agenda filters, such as
`GET /v1/scheduled-posts?status=scheduled` by default and adding
`run_id={run_id}`, `artifact_id={artifact_id}`, or `platform={platform}` when a
specific run, rendered artifact, or platform is selected. The optional
browser-local date/time range controls are converted to RFC3339 `from` and
`to` query parameters before refreshing the agenda. Choosing `all` or clearing
the range omits that query parameter so the API returns the complete local set
for that dimension.

It shows upcoming local schedule records grouped by browser-local calendar day
after the active filters are applied. Each day header shows the row count and
earliest scheduled time for that day, while rows stay ordered by `scheduled_at`
with schedule ID as a deterministic fallback. Rows include scheduled time,
platform, status, run ID/topic when the run list can resolve it, artifact ID
when present, and compact validation metadata when the API provides it.
Validation details stay collapsed inside each row so warnings such as a
missing selected artifact or external-only credential guardrails are available
without crowding normal agenda scanning. Missing or malformed validation
metadata is ignored by the view model. Agenda loading errors stay isolated from
the selected run review state so operators can keep inspecting run details even
if the scheduled-post list fails.

The agenda defaults to all runs, all artifacts, and the `scheduled` status so
active upcoming local records stay in focus. Run, artifact, status, and
platform filters are sent to the API and then applied again in the browser as a
defensive display fallback. Artifact filter options come from the currently
selected run's rendered artifacts; stale artifact selections are reset to all
artifacts when run details change. Operators can switch status to `all` or
`cancelled` to inspect cancelled local records without changing the selected
run or run detail review state.

Agenda rows with `scheduled` status can be marked `cancelled` from the Web UI.
The action sends:

```text
PATCH /v1/scheduled-posts/{schedule_id}
{"status":"cancelled"}
```

After a successful cancellation the UI refreshes both the agenda and selected
run detail. Cancelled rows remain visible with their updated status and do not
offer a repeated cancel action. This is still local metadata only and does not
contact or undo external platform scheduling.

## Artifact Browser

The artifact section reads artifact metadata from the selected run detail
payload and uses the existing local-only file endpoint for open/download links:

```text
GET /v1/runs/{run_id}/artifacts/{artifact_id}/file
```

Image and video artifacts render inline previews. Text and JSON artifacts keep a
compact metadata row with filename, path, timestamp, checksum status, and direct
links.

Artifact cards also summarize local schedule usage from the selected run detail
payload. A card shows when its artifact ID appears on one or more schedule
records, including counts by status and either the next future scheduled time or
the latest schedule time. Schedule rows without an artifact ID are ignored so
artifact-less records do not attach to unrelated files.

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
