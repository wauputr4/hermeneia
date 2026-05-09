# MVP Roadmap

## Phase 0 — Project Scaffold

Goal:

- Define project name, vision, stack, architecture, and open-source direction.

Status: in progress.

Phase 0 must also lock the MVP definition and explicit non-goals. See [MVP Definition and Non-Goals](mvp.md).

## Phase 1 — CLI Content Run MVP

Goal:

Create content runs from topic/brief and store metadata in SQLite while keeping generated artifacts in run folders.

Commands:

```bash
hermeneia create --topic "latest AI news" --type carousel
hermeneia revise --run runs/example --instruction "Make it sharper"
hermeneia render --run runs/example
```

Deliverables:

- Go CLI scaffold.
- Content run folder generation.
- Brief JSON schema.
- SQLite-backed revision history.
- Deterministic run artifact folder.
- Render command that records exported artifact metadata.

## Phase 2 — Carousel Renderer MVP

Goal:

Generate PNG carousel slides from structured JSON and one template.

Deliverables:

- One carousel template.
- Image renderer pipeline.
- Output slide PNG files.

## Phase 3 — Remotion Short Video MVP

Goal:

Generate one 9:16 short video from scene JSON.

Deliverables:

- One Remotion template.
- MP4 export.
- CLI command to render video.

## Phase 4 — Web UI MVP

Goal:

Build a simple UI to view runs, edit briefs, pick templates, preview outputs, and export assets.

Initial API slice:

- `hermeneia serve` exposes the local JSON API.
- API handlers call the same Go workflow service as the CLI.
- Routes cover run create/list/show/delete, brief and artifact listing,
  revision, render, and research-run creation.

Deliverables:

- Go HTTP API for web and agents.
- SvelteKit app.
- Run list.
- Brief editor.
- Template picker.
- Preview/export page.

## Phase 5 — Research Automation

Goal:

Add trend discovery and AI-assisted content planning.

Deliverables:

- Source list.
- Research summaries.
- Content idea ranking.
- Brief generation from research.

Initial CLI slice:

- `hermeneia research` accepts curated source URLs with `--source`.
- The run folder stores inspectable `research.json`.
- SQLite records the research file as a `research_json` artifact.
- The first brief is generated from the research plan while preserving the normal revise/render workflow.

## Phase 6 — Scheduling and Publishing

Goal:

Support scheduled publishing to social platforms.

Deliverables:

- Content calendar.
- Schedule records in SQLite with post status tracking. (initial slice shipped)
- Meta API integration.
- YouTube integration.
- Platform connectors that keep credentials out of SQLite.
