# MVP Roadmap

## Phase 0 — Project Scaffold

Goal:

- Define project name, vision, stack, architecture, and open-source direction.

Status: in progress.

## Phase 1 — CLI Content Run MVP

Goal:

Create content runs from topic/brief and store them in file-based history.

Commands:

```bash
hermeneia create --topic "latest AI news" --type carousel
hermeneia revise --run runs/example --instruction "Make it sharper"
```

Deliverables:

- Go CLI scaffold.
- Content run folder generation.
- Brief JSON schema.
- Revision history file.

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

Deliverables:

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

## Phase 6 — Scheduling and Publishing

Goal:

Support scheduled publishing to social platforms.

Deliverables:

- Content calendar.
- Meta API integration.
- YouTube integration.
- Post status tracking.
