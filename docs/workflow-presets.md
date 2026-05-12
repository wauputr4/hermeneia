# Workflow Preset Authoring Guide

Workflow presets describe repeatable Hermeneia flows using existing service
steps. They are JSON contracts, not arbitrary scripts.

The current preset catalog is read-only. CLI, API, and Web UI surfaces can list
presets and use their metadata, but run creation does not execute a `workflow_id`
yet.

## Directory Structure

Built-in presets live under `workflows/`:

```text
workflows/
├─ simple-carousel.json
└─ research-carousel.json
```

Each file contains one preset. The `id` must be unique across the catalog.

## Preset Fields

Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Stable preset ID. |
| `name` | string | Display name for CLI, API, and Web UI catalogs. |
| `description` | string | Operational summary. |
| `content_type` | string | `carousel` or `short_video`. |
| `default_template_id` | string | Installed template ID used by default. |
| `steps` | array | Ordered step definitions. |
| `required_inputs` | string array | Inputs an operator or agent must supply. |

Optional field:

- `revision_policy`: object with `mode` and `max_revisions`.

Supported step types:

- `create_brief`
- `research_plan`
- `revise_brief`
- `render`
- `schedule_record`

Unknown step types are rejected. `default_template_id` must reference an
installed template whose `content_type` matches the preset.

## Simple Preset Example

Create `workflows/local-simple-carousel.json`:

```json
{
  "id": "local-simple-carousel",
  "name": "Local Simple Carousel",
  "description": "Create a carousel brief from a topic and render PNG slides.",
  "content_type": "carousel",
  "default_template_id": "carousel/local-clean",
  "steps": [
    {"type": "create_brief", "name": "Create brief"},
    {"type": "render", "name": "Render carousel"}
  ],
  "required_inputs": ["topic"],
  "revision_policy": {
    "mode": "manual",
    "max_revisions": 3
  }
}
```

The preset uses existing Hermeneia steps only. It does not run shell commands,
load plugins, or publish to external platforms.

## Validation Behavior

The workflow validator rejects:

- duplicate preset IDs,
- missing required fields,
- empty `required_inputs`,
- unsupported `content_type` values,
- unknown step types,
- missing `default_template_id` references,
- template references whose content type does not match the preset content type.

List the catalog with:

```bash
hermeneia workflows
```

Or through the local API:

```bash
curl http://127.0.0.1:19318/v1/workflows
curl http://127.0.0.1:19318/v1/workflows/local-simple-carousel
```

## Examples

- `examples/workflows/local-simple-carousel.json` contains a preset that pairs
  with `examples/templates/carousel/local-clean/template.json`.
- Built-in examples remain in `workflows/simple-carousel.json` and
  `workflows/research-carousel.json`.

## Non-Goals

Workflow presets are intentionally not:

- a general automation language,
- a visual DAG editor,
- a plugin system,
- a publishing connector,
- a place to store credentials or platform tokens.

External publishing connectors remain a later product slice after template,
workflow, and Web UI customization are stable.
