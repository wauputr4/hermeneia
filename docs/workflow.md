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

The MVP CLI revision path is deterministic. It records the revision instruction, creates the next brief version, and adds a visible revision note. Later AI-assisted revision can replace that placeholder while keeping the same append-only version contract.

## File Artifact Convention

Generated run files use this deterministic layout:

```text
runs/{run-id}/
├─ brief.v1.json
├─ brief.v2.json
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

## 7. Export

Supported MVP exports:

- PNG slides,
- MP4 video,
- caption text,
- content JSON.

## 8. Schedule / Publish

Later integrations:

- Meta / Instagram / Facebook,
- YouTube,
- TikTok,
- LinkedIn.
