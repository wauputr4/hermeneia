# Changelog

All notable changes to Hermeneia are documented here.

## v0.1.0 - 2026-05-09

### Added

- CLI-first MVP workflow with `init`, `create`, `research`, `list`, `show`, `revise`, `render`, and `serve`.
- SQLite-backed content runs, brief versions, revision events, render jobs, and artifact metadata.
- Inspectable run folders under `runs/{run-id}/` with brief JSON, research JSON, history, render input, captions, and exported assets.
- Deterministic carousel rendering for `carousel/ai-news-clean`.
- Local short-video MP4 render path for `video/ai-news-short`.
- Local JSON HTTP API on `127.0.0.1:19317` for agent and future UI workflows.
- Optional OpenAI Responses API research planner behind explicit `--planner openai` or API `planner: "openai"`.
- Open-source project hygiene: license, contribution guide, security policy, code of conduct, ADRs, MVP docs, API docs, and release process.

### Fixed

- API responses now return database-backed timestamps for newly created runs, briefs, research artifacts, and render artifacts.
- Render artifact hydration now avoids N+1 metadata reads.
- Post-render metadata recording is isolated from request cancellation so completed render files and SQLite artifact rows stay aligned.

### Notes

- This is a local-first technical MVP, not a hosted multi-user product.
- OpenAI configuration is optional; default research planning remains deterministic and local.
- Video rendering requires `ffmpeg` on `PATH`.
- Publishing/scheduling integrations and the SvelteKit web UI are intentionally deferred.
