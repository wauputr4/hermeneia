# Schemas

Hermeneia uses structured JSON documents at workflow boundaries so content runs remain inspectable, revision-friendly, and renderer-agnostic.

## Content Brief

The MVP content brief is the planning document stored in SQLite as `brief_versions.body_json` and mirrored in run folders as files like `runs/{run-id}/brief.v1.json`.

### JSON Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `topic` | string | yes | Subject of the content run. |
| `angle` | string | yes | Editorial framing for the topic. |
| `hook` | string | yes | Opening idea intended to stop the scroll. |
| `target_audience` | string | yes | Primary audience for the content. |
| `platform` | string | yes | Target platform, such as `instagram`, `linkedin`, `tiktok`, or `youtube_shorts`. |
| `content_type` | string | yes | Output type, such as `carousel` or `short_video`. |
| `tone` | string | yes | Voice and editorial style guidance. |
| `key_points` | array of strings | yes | Main points the generated content should cover. |
| `visual_direction` | string | yes | Visual style and layout guidance for templates or renderers. |
| `cta` | string | yes | Call to action. |
| `caption_draft` | string | yes | Initial social caption text. |
| `hashtags` | array of strings | yes | Suggested hashtags, including the `#` prefix. |

### Go Type

The canonical MVP Go representation lives in `internal/brief.Brief`.

### Example

See [brief.ai-agents-carousel.json](../examples/brief.ai-agents-carousel.json).
