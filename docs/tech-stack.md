# Technology Stack

## Recommendation

Use a hybrid stack:

```text
Backend / CLI: Go
Frontend: SvelteKit + TypeScript
Video Rendering: Remotion + React + TypeScript
Image Rendering: HTML/CSS/SVG pipeline + optional AI image generation
Storage MVP: SQLite
Storage later: PostgreSQL if needed
```

## Why Go for Backend and CLI

Go is a strong fit for Hermeneia because it provides:

- simple syntax,
- fast execution,
- easy deployment as a single binary,
- strong concurrency primitives,
- reliable HTTP APIs,
- maintainable CLI development,
- good long-term scalability.

Go should own:

- API server,
- CLI commands,
- workflow orchestration,
- file/database storage,
- job queue logic,
- publishing integrations later.

## Why TypeScript Still Matters

TypeScript is required for the frontend and Remotion ecosystem.

TypeScript should own:

- SvelteKit UI,
- React/Remotion video templates,
- browser previews,
- template editing interfaces.

## Rendering Boundary

The Go backend should call rendering workers through a clear boundary:

```text
Go workflow service
→ structured JSON input
→ TypeScript renderer worker
→ PNG/MP4 output
```

This keeps the workflow reliable while still taking advantage of React and Remotion for media generation.

## MVP Storage

Start with SQLite as the durable local database, while keeping exported run artifacts in folders:

```text
runs/{run-id}/
├─ brief.v1.json
├─ brief.v2.json
├─ content.json
├─ history.md
└─ output/
```

Later, move to SQLite or PostgreSQL when the UI needs filtering, collaboration, scheduling, and audit logs.
