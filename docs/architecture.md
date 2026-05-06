# Architecture

Hermeneia should be designed as a modular system with clear boundaries between workflow orchestration, content generation, rendering, storage, and publishing.

## Proposed Monorepo Layout

```text
hermeneia/
├─ apps/
│  ├─ api/             # Go backend API and workflow service
│  ├─ cli/             # Go CLI entrypoint
│  └─ web/             # SvelteKit web UI
├─ packages/
│  ├─ renderer-video/  # Remotion renderer
│  ├─ renderer-image/  # Carousel/image renderer
│  ├─ prompts/         # Prompt templates and content schemas
│  └─ shared/          # Shared schemas/types where needed
├─ templates/
│  ├─ carousel/
│  └─ video/
├─ docs/
├─ examples/
└─ runs/               # Local generated content runs; ignored later
```

## Main Components

### API / Workflow Service

Written in Go.

Responsibilities:

- Manage content runs.
- Store brief versions and revision history.
- Trigger research, generation, rendering, and export jobs.
- Expose HTTP API for the web UI and automation.

### CLI

Written in Go.

Responsibilities:

- Run the same workflow from terminal.
- Allow AI agents to operate Hermeneia.
- Create, revise, render, and export content runs.

### Web UI

Written with SvelteKit and TypeScript.

Responsibilities:

- Review content ideas.
- Edit briefs.
- Select templates.
- Preview generated assets.
- Track revision history.
- Export or schedule content.

### Video Renderer

Built with Remotion.

Responsibilities:

- Render short videos from structured scene JSON.
- Provide reusable video templates.
- Output MP4 for social platforms.

### Image Renderer

Initial options:

- HTML/CSS templates rendered to PNG through Playwright.
- SVG templates rendered to PNG.
- AI image generation/editing for backgrounds or references.

## Design Rule

Business workflow logic should live in the Go backend/core, not inside the UI or renderer.

Renderers should be deterministic workers that receive structured input and output assets.
