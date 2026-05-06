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
