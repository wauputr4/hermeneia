# Workflow

Hermeneia's workflow should be simple and inspectable.

## Content Run Lifecycle

```text
1. Research
2. Plan / Brief
3. Template Selection
4. Generation
5. Rendering
6. Revision
7. Export
8. Schedule / Publish
```

## 1. Research

Research can come from:

- web search,
- RSS feeds,
- curated source lists,
- manually provided URLs,
- AI agent summaries.

Output should be a structured research summary.

The MVP CLI starts with manually supplied source URLs:

```bash
hermeneia research --topic "AI agents in marketing" --source "https://example.com/ai-agents"
```

This writes `research.json` into the run folder, preserves every source URL for traceability, and generates the first schema-valid brief draft from the research plan. By default the planner is deterministic. When `OPENAI_API_KEY` and `OPENAI_MODEL` are configured, `hermeneia research --planner openai` can use the OpenAI Responses API to summarize the supplied source metadata and rank ideas while preserving the same stored research contract.

## 2. Brief

A brief should contain:

- topic,
- angle,
- hook,
- audience,
- key points,
- tone,
- visual direction,
- platform,
- content type.

The canonical MVP JSON shape is documented in [Schemas](schemas.md), with an example at [examples/brief.ai-agents-carousel.json](../examples/brief.ai-agents-carousel.json).

## 3. Template Selection

Users or agents select a template:

```text
carousel/ai-news-clean
video/ai-news-short
```

## 4. Generation

The system converts the brief into final structured content.

Carousel example:

```json
{
  "slides": [
    {"type": "cover", "headline": "AI Agents Are Changing Marketing"},
    {"type": "point", "title": "What changed?", "body": "..."}
  ],
  "caption": "...",
  "hashtags": ["#AI", "#PenaDigital"]
}
```

Video example:

```json
{
  "scenes": [
    {"duration": 3, "text": "AI agents just got a major upgrade", "visual": "futuristic interface"}
  ],
  "caption": "..."
}
```

## 5. Rendering

Carousel:

```text
structured content → image renderer → PNG slides
```

Short video:

```text
structured scenes → Remotion renderer → MP4
```

The CLI MVP keeps renderer boundaries explicit:

- Go workflow code builds structured content from the latest brief version.
- Renderer code receives structured JSON, never raw prompts.
- SQLite stores artifact metadata and checksums.
- Files remain inspectable under the run directory.

The first carousel renderer is a deterministic Go PNG renderer for `carousel/ai-news-clean`.
The first video path writes a Remotion-ready scene contract to `output/video/remotion-input.json` and creates a local MP4 output for the MVP loop. The Remotion composition scaffold in `packages/renderer-video` consumes the same contract.

## 6. Revision

Every revision should create a new version rather than overwriting previous work.

```text
brief.v1.json
brief.v2.json
history.md
```

Custom instructions enter the MVP workflow through revision commands:

```bash
hermeneia revise <run-id> --instruction "Make the hook sharper and more practical"
```

In the current CLI MVP, this instruction is recorded deterministically in SQLite and `history.md`, reflected into the next brief version, and never sent to OpenAI. Future AI-assisted revision can replace that deterministic placeholder while keeping the same append-only version contract.

Optional future OpenAI configuration:

```text
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=
```

These variables are intentionally optional for the MVP. They should be required only when commands actually call OpenAI for research, brief generation, or AI-assisted revision.

## File Artifact Convention

Generated run files use this deterministic layout:

```text
runs/{run-id}/
├─ brief.v1.json
├─ brief.v2.json
├─ research.json
├─ content.json
├─ history.md
└─ output/
   ├─ carousel/
   │  ├─ slide-01.png
   │  ├─ slide-02.png
   │  └─ caption.txt
   └─ video/
      ├─ remotion-input.json
      └─ ai-news-short.mp4
```

Versioning rules:

- Brief revisions are append-only as `brief.v{n}.json`.
- `history.md` records create, revise, and render events.
- `content.json` represents the latest structured render input for the run.
- Render outputs live under content-type-specific output folders.
- SQLite stores artifact rows with kind, path, checksum, run id, and brief version id.

The SvelteKit web UI follows the same lifecycle through the Go API: create a
run, inspect brief versions, submit revision instructions, trigger rendering,
and review artifact metadata. The CLI remains the primary automation surface,
while the UI provides a local review console for humans.

## Workflow Presets

Workflow presets are JSON contracts for repeatable flows that map to existing
Hermeneia service steps such as `create_brief`, `research_plan`, and `render`.
They do not run arbitrary scripts. The initial contract and built-in presets are
documented in [Workflow Presets](workflows.md).

## 7. Export

Supported MVP exports:

- PNG slides,
- MP4 video,
- caption text,
- content JSON.

## 8. Schedule / Publish

The first scheduling slice records future publishing slots and status tracking
without calling external social APIs. It rejects past schedule timestamps,
validates known platform names, and stores schedule metadata in SQLite, but
platform credentials must remain in secret managers or platform connectors
outside the Hermeneia database.

```bash
hermeneia schedule <run-id> --platform instagram --at 2026-05-10T02:00:00Z
hermeneia schedules
```

Supported planned platforms:

- Meta / Instagram / Facebook,
- YouTube,
- TikTok,
- LinkedIn.
