# HTTP API

Hermeneia exposes a small local JSON API for web UI and agent workflows. The
API is intentionally thin: handlers decode HTTP requests and call the same Go
workflow service used by the CLI.

Run it locally with:

```bash
hermeneia serve --addr 127.0.0.1:8080
```

The server uses the same configuration as the CLI:

- `HERMENEIA_DATABASE_PATH` controls the SQLite database path.
- Run files are written under `runs/` by default.

## Conventions

- Request and response bodies are JSON.
- Timestamps use Go's standard RFC3339 JSON encoding.
- Error responses use:

```json
{"error":"message"}
```

## Routes

### Health

```http
GET /health
```

Returns:

```json
{"status":"ok"}
```

### List Runs

```http
GET /v1/runs
```

Returns:

```json
{
  "runs": [
    {
      "id": "run-20260509-ai-agents-a1b2c3",
      "topic": "AI agents in marketing",
      "content_type": "carousel",
      "template_id": "carousel/ai-news-clean",
      "created_at": "2026-05-09T00:00:00Z"
    }
  ]
}
```

### Create Run

```http
POST /v1/runs
```

Request:

```json
{
  "topic": "AI agents in marketing",
  "content_type": "carousel",
  "template_id": "carousel/ai-news-clean",
  "tone": "clear and practical",
  "platform": "instagram",
  "target_audience": "content operators"
}
```

Returns `201 Created` with the created run, first brief, and runfile paths.

### Create Research Run

```http
POST /v1/research-runs
```

Request:

```json
{
  "topic": "AI agents in marketing",
  "content_type": "carousel",
  "sources": [
    {
      "url": "https://example.com/ai-agents",
      "title": "Agent workflows",
      "note": "Seed source for editorial review"
    }
  ]
}
```

Returns `201 Created` with the run, first brief, `research.json` path, and
research artifact metadata.

### Show Run

```http
GET /v1/runs/{run_id}
```

Returns the run, brief versions, revision events, and artifact metadata.

### Delete Run

```http
DELETE /v1/runs/{run_id}
```

Deletes the content run metadata and its local run folder. Returns `204 No
Content`.

### List Brief Versions

```http
GET /v1/runs/{run_id}/briefs
```

Returns all brief versions for the run.

### List Artifacts

```http
GET /v1/runs/{run_id}/artifacts
```

Returns artifact rows for the run.

### Revise Run

```http
POST /v1/runs/{run_id}/revisions
```

Request:

```json
{
  "instruction": "Make the hook sharper"
}
```

Returns `201 Created` with the previous brief and the new brief version.

### Render Run

```http
POST /v1/runs/{run_id}/render
```

Renders the latest brief version using the run's content type and template.
Returns `201 Created` with structured content and created artifact metadata.
