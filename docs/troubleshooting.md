# Troubleshooting Notes

This file should be updated after meaningful implementation work.

## Early Considerations

### Remotion License

Remotion has special license terms. It is free for individuals, small for-profit organizations up to certain limits, non-profits, and evaluation use cases. Before commercial production use, review Remotion's official license terms.

### Renderer Boundary

Go should not directly own React rendering logic. Use structured JSON as the boundary between Go workflow services and TypeScript renderers.

### Revision History

Avoid overwriting generated content during revisions. Store versions explicitly.

### Social Platform APIs

Publishing integrations should be added later because Meta, YouTube, and TikTok APIs introduce authentication, rate limits, review requirements, and platform-specific media constraints.

## 2026-05-07 — MVP Scope Locked

Added `docs/mvp.md` to define the first usable Hermeneia MVP and explicit non-goals.

Important direction:

- MVP prioritizes CLI + SQLite + structured brief/revision workflow.
- Web UI, full AI research automation, and publishing integrations are intentionally deferred.
- Generated artifacts may live in `runs/`, but SQLite should track metadata and revision history.
