# ADR 0001: Use Go for Backend and CLI

## Status

Accepted

## Context

Hermeneia needs a reliable workflow engine that can run locally, expose a CLI
for agents and power users, and later serve an HTTP API for the web UI. The
MVP should stay easy to install and operate without requiring a large runtime.

## Decision

Use Go for the backend workflow service, CLI entrypoint, storage layer, and HTTP
API.

## Consequences

- Hermeneia can ship as a small single binary for local MVP usage.
- The CLI and HTTP API can share the same service layer and persistence code.
- Long-running jobs, rendering orchestration, and future publishing workers can
  use Go's standard concurrency primitives.
- Frontend and Remotion code remain TypeScript-based, so schema boundaries must
  stay explicit.

## Alternatives Considered

- Node.js for the full stack: would simplify sharing TypeScript types but makes
  the local workflow binary and long-running backend less self-contained.
- Python for workflow scripts: would be quick for prototypes but less ideal for
  a durable CLI/API binary.
- Rust for backend and CLI: strong systems choice, but higher implementation
  cost for the current MVP scope.
