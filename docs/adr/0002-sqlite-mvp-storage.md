# ADR 0002: Use SQLite for MVP Storage

## Status

Accepted

## Context

The first MVP is local-first and CLI-first. It needs durable metadata for runs,
brief versions, render jobs, artifact references, and revision history without
introducing hosted infrastructure too early.

## Decision

Use SQLite as the MVP metadata store. Keep generated asset bytes and inspectable
JSON/Markdown snapshots in `runs/{run-id}/`, with SQLite rows pointing to those
files.

## Consequences

- The MVP works locally with no database server.
- Tests can exercise real persistence behavior with temporary SQLite files.
- Metadata remains queryable while exported files stay easy to inspect.
- Hosted collaboration may require a later PostgreSQL migration or adapter.

## Alternatives Considered

- PostgreSQL from day one: better for hosted collaboration but unnecessary
  operational weight for a local MVP.
- File-only storage: simple, but harder to query, list, and relate versions,
  artifacts, and render jobs consistently.
- Embedded key-value stores: lightweight, but less transparent than SQL for the
  relational metadata Hermeneia tracks.
