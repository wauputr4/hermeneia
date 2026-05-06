# MVP Definition and Non-Goals

This document defines the first usable Hermeneia MVP.

The MVP should be intentionally small. Its purpose is to prove that Hermeneia can turn a structured content idea into real exportable social media assets through a reproducible workflow.

## MVP Mission

The first MVP must answer one question:

> Can Hermeneia create, revise, and export a template-driven content run from the command line?

The answer should be demonstrated with at least one carousel output and one short-video output path, even if the first renderers are minimal.

## MVP User

The MVP is designed for:

- content operators,
- AI agents,
- developers,
- and small teams that need a repeatable content production workflow.

The initial user does not need a full web dashboard. The CLI is enough for the first working loop.

## MVP Workflow

```text
Create content run
→ store brief in SQLite
→ select template
→ generate structured content
→ render/export artifact
→ revise into a new version
→ inspect history
```

## MVP Must-Haves

### 1. Go CLI

The CLI must support the first workflow loop.

Required commands:

```bash
hermeneia init
hermeneia create
hermeneia list
hermeneia show
hermeneia revise
```

Render commands can arrive after the storage and brief workflow are stable.

### 2. SQLite Storage

SQLite is the MVP source of truth for metadata.

It must store:

- content runs,
- brief versions,
- selected content type,
- selected template,
- revision events,
- artifact references.

Generated files can still live in `runs/`, but SQLite should track them.

### 3. Structured Brief Schema

The MVP must use structured JSON instead of raw prompt-only output.

A content brief should include:

- topic,
- angle,
- hook,
- audience,
- platform,
- content type,
- tone,
- key points,
- visual direction,
- CTA,
- caption draft,
- hashtags.

### 4. Revision History

Revisions must not overwrite previous versions.

Each revision should create a new version and record:

- revision instruction,
- timestamp,
- previous version,
- new version,
- changed fields when available.

### 5. File Artifact Convention

The MVP must define where generated files are stored.

```text
runs/{run-id}/
├─ brief.v1.json
├─ brief.v2.json
├─ content.json
├─ history.md
└─ output/
```

### 6. First Carousel Output Path

The MVP should support a simple path to carousel output.

The first version may use a minimal renderer as long as it produces inspectable output and keeps the template contract clear.

### 7. First Short Video Output Path

The MVP should define the Remotion input contract and produce a first short-video render path once the core workflow is stable.

## MVP Success Criteria

The MVP is successful when a user or AI agent can run a flow similar to:

```bash
hermeneia init
hermeneia create --topic "AI agents in marketing" --type carousel --template carousel/ai-news-clean
hermeneia show <run-id>
hermeneia revise <run-id> --instruction "Make the hook sharper"
hermeneia list
```

And the project stores:

- a SQLite record for the run,
- at least two brief versions after revision,
- a history entry,
- a deterministic run artifact folder.

## Explicit Non-Goals for MVP

The MVP will not include:

- full AI research automation,
- automatic trend ranking,
- production-ready social media publishing,
- Meta/Instagram/Facebook API integration,
- YouTube API integration,
- TikTok API integration,
- multi-user collaboration,
- cloud deployment,
- role-based access control,
- payment or billing,
- advanced template marketplace,
- complete web dashboard,
- real-time collaborative editing.

These are future phases after the CLI and core workflow prove useful.

## Phase Boundary

MVP work should prioritize output over completeness.

The correct order is:

1. CLI and SQLite foundation.
2. Brief schema and revision history.
3. Deterministic artifact storage.
4. First carousel output.
5. First Remotion video output.
6. Web UI and AI research later.

## Quality Bar

Even the MVP should maintain open-source quality:

- clear documentation,
- tests for meaningful code changes,
- no committed secrets,
- deterministic examples,
- explicit limitations,
- readable CLI errors.
