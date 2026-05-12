# Template Authoring Guide

This guide explains how to create a local Hermeneia template without reading the
Go renderer internals.

Templates are manifest-backed media contracts. A template tells Hermeneia which
content type it supports, which renderer owns the output, which artifact kinds
will be produced, and what structured input must validate before rendering.

## Directory Structure

Custom template roots use the same structure as the built-in `templates/`
directory:

```text
my-templates/
└─ carousel/
   └─ local-clean/
      ├─ template.json
      └─ preview.png
```

The manifest path defines the template ID:

```text
carousel/local-clean -> my-templates/carousel/local-clean/template.json
```

Point Hermeneia at one or more custom roots with `HERMENEIA_TEMPLATE_PATH`.
Use the operating system path-list separator (`:` on Linux/macOS, `;` on
Windows):

```bash
HERMENEIA_TEMPLATE_PATH=/absolute/path/to/my-templates hermeneia templates
```

Built-in templates load first. Custom roots then load in
`HERMENEIA_TEMPLATE_PATH` order. Duplicate IDs fail validation instead of
overriding earlier templates.

## Manifest Fields

Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Must match the manifest path below the template root. |
| `name` | string | Display name for CLI, API, and Web UI catalogs. |
| `content_type` | string | `carousel` or `short_video`. |
| `description` | string | Short operational summary. |
| `version` | string | Template contract version. |
| `aspect_ratio` | string | Output format such as `4:5` or `9:16`. |
| `renderer` | string | Renderer key understood by Hermeneia. |
| `output_kinds` | string array | Artifact kinds expected from rendering. |
| `input_schema` | object | JSON Schema-style subset for render input. |

Optional fields:

- `preview_asset`: relative preview image path.
- `assets`: relative asset paths used by the template.

Asset paths must stay inside the template directory. Absolute paths and `../`
traversal are rejected.

## Input Schema

Hermeneia validates manifest `input_schema` before run creation and before
rendering. The supported subset is deliberately small:

- `type`: `object`, `array`, `string`, `integer`, `number`, or `boolean`
- `required`
- `properties`
- `items`, `minItems`, `maxItems`
- `const`
- `enum`
- `minimum`

Unsupported schema keywords are rejected so authors do not assume an unenforced
rule is active.

## Carousel Template From Scratch

Create `my-templates/carousel/local-clean/template.json`:

```json
{
  "id": "carousel/local-clean",
  "name": "Local Clean Carousel",
  "content_type": "carousel",
  "description": "A local 4:5 carousel template for concise explainers.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["content_json", "carousel_png", "caption_text"],
  "input_schema": {
    "type": "object",
    "required": ["template", "slides", "caption", "hashtags"],
    "properties": {
      "template": {"const": "carousel/local-clean"},
      "slides": {
        "type": "array",
        "minItems": 1,
        "maxItems": 10,
        "items": {
          "type": "object",
          "required": ["type", "headline", "body"],
          "properties": {
            "type": {"enum": ["cover", "point", "closing"]},
            "headline": {"type": "string"},
            "body": {"type": "string"}
          }
        }
      },
      "caption": {"type": "string"},
      "hashtags": {
        "type": "array",
        "items": {"type": "string"}
      }
    }
  },
  "preview_asset": "preview.png",
  "assets": []
}
```

Validate it through the catalog:

```bash
HERMENEIA_TEMPLATE_PATH=/absolute/path/to/my-templates hermeneia templates
```

Create a run with the custom template:

```bash
HERMENEIA_TEMPLATE_PATH=/absolute/path/to/my-templates \
  hermeneia create --topic "AI agents for small teams" --type carousel --template carousel/local-clean
```

Render uses the same schema again. If the brief produces a payload that does
not match the manifest, `hermeneia render` fails before writing partial output.

## Examples

- `examples/templates/carousel/local-clean/template.json` contains a complete
  custom carousel manifest.
- Copy the `examples/templates` directory outside the repository, then point
  `HERMENEIA_TEMPLATE_PATH` at that copied root.

## Troubleshooting

- Duplicate ID: rename the custom `id` and directory, or remove the conflicting
  root from `HERMENEIA_TEMPLATE_PATH`.
- Invalid content type: use only `carousel` or `short_video`.
- Missing asset: keep `preview_asset` and `assets` relative to the template
  directory and commit or copy the referenced files.
- ID/path mismatch: ensure `carousel/local-clean` lives at
  `carousel/local-clean/template.json`.
- Empty custom root: Hermeneia ignores roots with no `template.json` files.
  Current releases still require the built-in `templates/` directory to be
  discoverable before custom roots are merged.
