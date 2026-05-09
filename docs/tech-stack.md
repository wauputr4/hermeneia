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

The first SvelteKit app is in `apps/web`. It uses the Go HTTP API through a
small TypeScript client and keeps testable display logic in small helpers so UI
state does not replace workflow-service behavior.

## Rendering Boundary

The Go backend should call rendering workers through a clear boundary:

```text
Go workflow service
→ structured JSON input
→ TypeScript renderer worker
→ PNG/MP4 output
```

This keeps the workflow reliable while still taking advantage of React and Remotion for media generation.

## OpenAI Configuration

The CLI MVP is usable without an OpenAI API key. It records custom revision instructions and creates deterministic draft briefs locally.

Future AI-assisted research, brief generation, and revision commands should read OpenAI configuration from environment variables instead of hardcoding secrets:

```text
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=
```

Guidelines:

- Keep real API keys in local `.env` files or secret managers.
- Use `hermeneia research --planner openai` only when both `OPENAI_API_KEY` and `OPENAI_MODEL` are set; otherwise the CLI stays on the deterministic local planner.
- Commit only placeholder values in `.env.example`.
- Treat `OPENAI_BASE_URL` as optional for the official OpenAI API and useful for compatible gateways.
- Keep prompt input/output inspectable by storing generated briefs and revisions as normal Hermeneia artifacts.

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

Later, keep SQLite for local/single-user installs or move to PostgreSQL when the UI needs hosted collaboration, scheduling, and audit logs.
