# Workflow Presets

Workflow presets define repeatable content production flows without introducing
arbitrary scripting or a visual graph builder.

Preset files are JSON documents under `workflows/`. They describe which
existing Hermeneia service steps should run, not custom code.

## Contract

Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Stable preset ID. |
| `name` | string | Human-readable display name. |
| `description` | string | Operational summary. |
| `content_type` | string | `carousel` or `short_video`. |
| `default_template_id` | string | Installed template ID used by default. |
| `steps` | array | Ordered workflow steps. |
| `required_inputs` | string array | Inputs an operator or agent must supply. |

Optional fields:

- `revision_policy.mode`
- `revision_policy.max_revisions`

Supported step types:

- `create_brief`
- `research_plan`
- `revise_brief`
- `render`
- `schedule_record`

Unknown step types are rejected. Presets may only reference installed templates
whose manifest `content_type` matches the preset `content_type`.

## Built-In Presets

### Simple Carousel

File: `workflows/simple-carousel.json`

```json
{
  "id": "simple-carousel",
  "content_type": "carousel",
  "default_template_id": "carousel/ai-news-clean",
  "steps": [
    {"type": "create_brief"},
    {"type": "render"}
  ],
  "required_inputs": ["topic"]
}
```

### Research to Carousel

File: `workflows/research-carousel.json`

```json
{
  "id": "research-carousel",
  "content_type": "carousel",
  "default_template_id": "carousel/ai-news-clean",
  "steps": [
    {"type": "research_plan"},
    {"type": "create_brief"},
    {"type": "render"}
  ],
  "required_inputs": ["topic", "sources"]
}
```

## Validation

The `internal/workflows` validator rejects:

- duplicate preset IDs,
- missing required preset fields,
- unsupported `content_type` values,
- unknown step types,
- missing `default_template_id` references,
- template references whose `content_type` does not match the preset.

Existing manual CLI and HTTP API operations remain unchanged. Presets are a
contract layer for upcoming CLI/API catalog exposure and UI workflow selection.

## Catalog Surfaces

Workflow presets are discoverable through:

- CLI: `hermeneia workflows`
- HTTP API: `GET /v1/workflows`
- HTTP API: `GET /v1/workflows/{workflow_id}`

Catalog responses include preset IDs, content types, default template IDs,
ordered step definitions, and required input names. Run creation does not accept
`workflow_id` yet; execution remains a later slice so this catalog API can stay
read-only and stable first.
