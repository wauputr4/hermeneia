# Template System

Templates are reusable media layouts that transform structured content into final assets.

## Template Types

```text
templates/
├─ carousel/
└─ video/
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

This keeps rendering deterministic and easier to revise.

## AI Image Generation

AI image generation or editing should be used as a supporting step:

- background generation,
- visual references,
- scene illustration,
- style variations.

The final template should still control layout and brand consistency.
