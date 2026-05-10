# Hermeneia

**Hermeneia** is an open-source AI-assisted content workflow engine for turning research into publishable social media assets.

It helps creators and teams move from trend discovery to content briefs, template-based carousel or video generation, revision history, export, and eventually scheduled publishing.

## Philosophical Meaning

The name **Hermeneia** is rooted in the idea of interpretation. It evokes the act of translating signals into meaning: taking scattered information, cultural context, trends, and raw research, then shaping them into narratives people can understand.

In this project, Hermeneia represents a creative bridge between:

- research and expression,
- signal and story,
- idea and artifact,
- human editorial judgment and AI-assisted production.

Hermeneia should not be a black-box content machine. It should be an interpretive workflow system: transparent, revisable, template-driven, and grounded in human intent.

## Vision

Hermeneia aims to become a practical content operating system:

```text
Research → Brief → Template → Generate → Revise → Export → Schedule
```

The first goal is simple: produce usable outputs quickly.

- Carousel posts from structured content briefs.
- Short videos from the same brief using Remotion templates.
- SQLite-backed metadata and revision history.
- CLI-first workflow so AI agents can operate it.
- Web UI later for human-friendly content operations.

## MVP Goal

The MVP scope is documented in [docs/mvp.md](docs/mvp.md).

The first MVP should generate real content assets from a topic or brief:

```text
Input: topic or research brief
Output: carousel PNG slides and/or short MP4 video
```

## Planned Interfaces

Hermeneia should be both:

1. **CLI-able** — for AI agents, automation, and power users.
2. **UI-based** — for content teams to review, revise, and export.

Example future CLI:

```bash
hermeneia research --topic "latest AI news"
hermeneia create --type carousel --template ai-news-clean --topic "AI agents in marketing"
hermeneia render --run runs/2026-05-07-ai-agents
hermeneia revise --run runs/2026-05-07-ai-agents --instruction "Make the hook sharper"
```

## Recommended Tech Direction

Hermeneia should use a pragmatic split:

- **Backend / workflow engine:** Go
- **Frontend:** SvelteKit + TypeScript
- **Video renderer:** Remotion + React + TypeScript
- **Carousel renderer:** HTML/CSS templates rendered to images, or SVG/PNG pipeline
- **Storage MVP:** SQLite
- **Storage later:** PostgreSQL if collaboration or hosted scale requires it

Go is recommended for the backend because it is fast, maintainable, easy to deploy, and excellent for building reliable CLIs, APIs, workers, and workflow services.

TypeScript remains important because Remotion and web UI workflows are naturally React/JS-based.

## Repository Status

This repository has a CLI-first MVP workflow foundation:

- create a deterministic content run from a topic,
- store the brief and revision history in SQLite,
- mirror inspectable files under `runs/{run-id}/`,
- render carousel PNG slides from structured content,
- render a short-video MP4 path from structured scene JSON,
- record generated artifact metadata and checksums in SQLite,
- review runs, brief versions, revisions, and artifacts through the first SvelteKit web UI slice.

## Local Development

Hermeneia is currently CLI-first. The Go entrypoint lives at `cmd/hermeneia`.

Prerequisites:

- Go toolchain compatible with the version in `go.mod`.
- `ffmpeg` on `PATH` when rendering the temporary local MP4 video output.

Common commands:

```bash
go test ./...
go run ./cmd/hermeneia help
go run ./cmd/hermeneia init
go run ./cmd/hermeneia create --topic "AI agents in marketing" --type carousel
go run ./cmd/hermeneia research --topic "AI agents in marketing" --source "https://example.com/ai-agents"
go run ./cmd/hermeneia research --topic "AI agents in marketing" --planner openai --source "https://example.com/ai-agents"
go run ./cmd/hermeneia templates
go run ./cmd/hermeneia revise <run-id> --instruction "Make the hook sharper"
go run ./cmd/hermeneia render <run-id>
go run ./cmd/hermeneia schedule <run-id> --platform instagram --at 2026-05-10T02:00:00Z
go run ./cmd/hermeneia schedules
go run ./cmd/hermeneia show <run-id>
```

Web UI development:

```bash
go run ./cmd/hermeneia serve --addr 127.0.0.1:19318
cd apps/web
npm install
npm run dev
```

The SvelteKit app reads the local Go API from `PUBLIC_HERMENEIA_API_BASE` and defaults to `http://127.0.0.1:19318`. The first UI slice supports run creation, run detail review, brief version inspection, revision instructions, render/export triggering, and artifact previews.

By default, `hermeneia init` creates or migrates `data/hermeneia.db`. To use an isolated database path:

```bash
HERMENEIA_DATABASE_PATH=/tmp/hermeneia.db go run ./cmd/hermeneia init
```

The current MVP does not require an OpenAI API key; custom instructions are supported through the deterministic `revise` command listed above. Research runs default to a deterministic local planner unless `--planner openai` is requested explicitly.

Future AI-assisted commands should use the optional OpenAI variables declared in `.env.example`:

```text
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=
```

Release and deployment notes:

- [Changelog](CHANGELOG.md)
- [Deployment](docs/deployment.md)
- [Release process](docs/release.md)

Current CLI surface:

- `hermeneia init` initializes SQLite storage.
- `hermeneia create` creates a run, writes `brief.v1.json`, and stores SQLite metadata.
- `hermeneia research` creates a run from traceable source URLs, writes `research.json`, and generates a schema-valid brief draft.
- `hermeneia research --planner openai` uses the optional OpenAI Responses API planner when `OPENAI_API_KEY` and `OPENAI_MODEL` are configured.
- `hermeneia templates` lists the local manifest-backed template catalog.
- `hermeneia list` lists stored runs.
- `hermeneia show` displays run, version, revision, and artifact counts.
- `hermeneia revise` creates the next brief version and records a revision event. In the MVP it applies a deterministic revision note instead of calling OpenAI.
- `hermeneia render` writes `content.json`, generates output assets, and stores artifact references.
- `hermeneia schedule` records a future publishing slot with platform validation and no stored platform credentials.
- `hermeneia schedules` lists scheduled publishing records and their statuses.

Default MVP templates:

- `carousel/ai-news-clean`
- `video/ai-news-short`

## License

Apache-2.0
