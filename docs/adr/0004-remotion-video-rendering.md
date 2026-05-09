# ADR 0004: Use Remotion for Video Rendering

## Status

Accepted

## Context

Hermeneia needs a path to generate short-form social video from structured scene
data. The renderer should support template reuse, design iteration, and eventual
browser previews.

## Decision

Use Remotion with React and TypeScript for video rendering. The Go MVP writes a
Remotion-ready scene contract and uses a temporary local MP4 renderer path until
the TypeScript worker is fully wired into the CLI/API.

## Consequences

- Video templates can use familiar React component patterns.
- The same scene contract can support local rendering, previews, and future
  worker execution.
- Go stays responsible for workflow orchestration and artifact tracking.
- Local MP4 generation depends on `ffmpeg` until the Remotion worker is fully
  integrated.

## Alternatives Considered

- Direct ffmpeg-only composition: efficient, but difficult to maintain as
  visual templates become more expressive.
- Browser automation video capture: flexible but more brittle for production
  rendering.
- External video SaaS: fast to prototype but introduces vendor dependency and
  credential complexity before the core workflow is proven.
