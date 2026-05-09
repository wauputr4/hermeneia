# ADR 0003: Use SvelteKit for the Web UI

## Status

Accepted

## Context

After the CLI workflow is stable, Hermeneia needs a web UI for reviewing runs,
editing briefs, selecting templates, previewing outputs, and triggering renders.
The UI should stay secondary to the Go workflow service rather than becoming the
source of business logic.

## Decision

Use SvelteKit and TypeScript for the web UI.

## Consequences

- The UI can be lightweight, fast, and productive for form-heavy editing flows.
- TypeScript fits the web and Remotion ecosystem.
- The Go HTTP API must expose stable JSON contracts for the UI.
- Shared schema documentation becomes important because Go and TypeScript both
  consume structured workflow data.

## Alternatives Considered

- React/Next.js: strong ecosystem, but SvelteKit is a leaner fit for the planned
  dashboard and editor surfaces.
- Server-rendered Go templates: simpler deployment, but less suitable for rich
  previews, editors, and future template tooling.
- Desktop-only UI: avoids web hosting concerns but limits team collaboration and
  browser-based review workflows.
