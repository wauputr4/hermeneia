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
- File-based history for revisions.
- CLI-first workflow so AI agents can operate it.
- Web UI later for human-friendly content operations.

## MVP Goal

The MVP should generate real content assets from a topic or brief:

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

This repository is currently in planning/scaffolding mode.

## License

Apache-2.0
