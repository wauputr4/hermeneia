# ADR 0005: Use Structured JSON as the Renderer Boundary

## Status

Accepted

## Context

Hermeneia should be inspectable and revisable. Renderers should not depend on
raw prompts, hidden state, or provider-specific AI output. The same brief should
be convertible into carousel or video content in a deterministic way.

## Decision

Use structured JSON contracts between workflow logic and renderers. The Go
workflow service builds structured render inputs such as `content.json` and
Remotion scene JSON, then renderer workers consume those files to produce
artifacts.

## Consequences

- Generated content can be inspected, tested, and versioned before rendering.
- AI-assisted steps can be added later without making renderers prompt-aware.
- CLI, API, and future web UI can all reason about the same content contracts.
- Schema changes need migration and compatibility care as templates evolve.

## Alternatives Considered

- Raw prompt-to-renderer flow: faster initially but opaque and hard to debug.
- Renderer-specific ad hoc structs only: simple per renderer, but leads to
  duplicated contracts and harder cross-template reuse.
- Database-only render payloads: queryable, but less convenient for local file
  review and external worker handoff.
