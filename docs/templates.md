# Template System

Templates are reusable media layouts that transform structured content into final assets.

## Template Types

```text
templates/
├─ carousel/
│  └─ ai-news-clean/
│     └─ template.json
└─ video/
   └─ ai-news-short/
      └─ template.json
```

Template IDs map directly to their built-in manifest paths:

```text
carousel/ai-news-clean -> templates/carousel/ai-news-clean/template.json
video/ai-news-short    -> templates/video/ai-news-short/template.json
```

The built-in loader scans local `template.json` files deterministically and
rejects duplicate IDs, missing required fields, unsupported content types, and
ID/path mismatches.

Hermeneia exposes the same manifest-backed catalog through:

- CLI: `hermeneia templates`
- HTTP API: `GET /v1/templates`
- HTTP API: `GET /v1/templates/carousel/ai-news-clean`
- Web UI: the local create-run form fetches `GET /v1/templates`, filters by
  manifest `content_type`, and displays template metadata before creating a run.

API responses intentionally omit local manifest paths. Manifests are the source
of truth for template metadata; SQLite stores the selected template ID on runs
and keeps a lightweight template row for run metadata relationships.

## Manifest Contract

Hermeneia template manifests are JSON. Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Stable template ID that maps to the manifest path. |
| `name` | string | Human-readable template name. |
| `content_type` | string | `carousel` or `short_video`. |
| `description` | string | Short operational description. |
| `version` | string | Template contract version. |
| `aspect_ratio` | string | Output format such as `4:5` or `9:16`. |
| `renderer` | string | Renderer implementation key. |
| `output_kinds` | string array | Artifact kinds the template can produce. |
| `input_schema` | object | JSON Schema-style description of renderer input. |

Optional fields:

- `preview_asset`
- `assets`

Example:

```json
{
  "id": "carousel/ai-news-clean",
  "name": "AI News Clean Carousel",
  "content_type": "carousel",
  "description": "A clean editorial carousel for AI news and explainers.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["content_json", "carousel_png", "caption_text"],
  "input_schema": {
    "type": "object",
    "required": ["template", "slides", "caption", "hashtags"]
  },
  "preview_asset": "preview.png",
  "assets": []
}
```

## Carousel Template

A carousel template should define:

- slide size,
- typography,
- color system,
- layout rules,
- supported slide types,
- brand assets,
- safe areas.

Example slide types:

- cover,
- point,
- quote,
- comparison,
- stat,
- closing CTA.

MVP template:

- `carousel/ai-news-clean`

The first implementation lives in the Go renderer and writes `1080x1350` PNG slides. The template manifest is stored at `templates/carousel/ai-news-clean/template.json`.

Carousel input contract:

```json
{
  "template": "carousel/ai-news-clean",
  "slides": [
    {"type": "cover", "headline": "...", "body": "..."},
    {"type": "point", "headline": "...", "body": "..."},
    {"type": "closing", "headline": "...", "body": "..."}
  ],
  "caption": "...",
  "hashtags": ["#Hermeneia"]
}
```

## Video Template

A video template should define:

- aspect ratio,
- fps,
- duration rules,
- scene types,
- transitions,
- typography,
- background behavior,
- caption placement.

Remotion should power video templates.

MVP template:

- `video/ai-news-short`

The Go CLI writes the Remotion input contract to `runs/{run-id}/output/video/remotion-input.json` and produces `ai-news-short.mp4` for the MVP loop. A Remotion composition scaffold is available in `packages/renderer-video` and consumes the same scene JSON contract.

Video input contract:

```json
{
  "template": "video/ai-news-short",
  "aspect_ratio": "9:16",
  "fps": 30,
  "scenes": [
    {"duration_seconds": 3, "text": "...", "visual": "..."}
  ],
  "caption": "..."
}
```

## Template Input Contract

Templates should receive structured JSON, not raw prompts.

This keeps rendering deterministic and easier to revise. The manifest
`input_schema` documents that structured payload for CLI, API, Web UI, and
future custom template loaders.

## AI Image Generation

AI image generation or editing should be used as a supporting step:

- background generation,
- visual references,
- scene illustration,
- style variations.

The final template should still control layout and brand consistency.
