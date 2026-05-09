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
- record generated artifact metadata and checksums in SQLite.

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
go run ./cmd/hermeneia revise <run-id> --instruction "Make the hook sharper"
go run ./cmd/hermeneia render <run-id>
go run ./cmd/hermeneia show <run-id>
```

By default, `hermeneia init` creates or migrates `data/hermeneia.db`. To use an isolated database path:

```bash
HERMENEIA_DATABASE_PATH=/tmp/hermeneia.db go run ./cmd/hermeneia init
```

The current MVP does not require an LLM API key. Custom instructions are supported through deterministic revision commands:

```bash
go run ./cmd/hermeneia revise <run-id> --instruction "Make the hook sharper"
```

Future AI-assisted commands should use the optional variables declared in `.env.example`:

```text
HERMENEIA_LLM_PROVIDER
HERMENEIA_LLM_API_KEY
HERMENEIA_LLM_BASE_URL
HERMENEIA_LLM_MODEL
```

Current CLI surface:

- `hermeneia init` initializes SQLite storage.
- `hermeneia create` creates a run, writes `brief.v1.json`, and stores SQLite metadata.
- `hermeneia list` lists stored runs.
- `hermeneia show` displays run, version, revision, and artifact counts.
- `hermeneia revise` creates the next brief version and records a revision event.
- `hermeneia render` writes `content.json`, generates output assets, and stores artifact references.

Default MVP templates:

- `carousel/ai-news-clean`
- `video/ai-news-short`

## License

Apache-2.0
