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

## 6. Revision

Every revision should create a new version rather than overwriting previous work.

```text
brief.v1.json
brief.v2.json
render.v1/
render.v2/
history.md
```

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
