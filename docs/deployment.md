# Deployment

Hermeneia `v0.1.0` is designed as a local-first CLI and API binary.

## Build

```bash
go build -o dist/hermeneia ./cmd/hermeneia
```

## Configuration

Required:

- none for the deterministic local MVP workflow.

Optional:

```bash
HERMENEIA_DATABASE_PATH=/path/to/hermeneia.db
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=
```

OpenAI variables are only used when the research planner is explicitly set to `openai`.

## Local CLI Usage

```bash
export HERMENEIA_DATABASE_PATH="$PWD/data/hermeneia.db"
./dist/hermeneia init
./dist/hermeneia create --topic "AI agents in marketing" --type carousel
./dist/hermeneia render <run-id>
```

Generated run files are written under `runs/` by default.

## Local API Usage

```bash
export HERMENEIA_DATABASE_PATH="$PWD/data/hermeneia.db"
./dist/hermeneia serve
```

Default address:

```text
127.0.0.1:19317
```

Health check:

```bash
curl -sS http://127.0.0.1:19317/health
```

## Process Manager Example

For a small server, run the built binary under a process manager such as systemd, Supervisor, or a container runtime. The service must preserve:

- the SQLite database path,
- the working directory or run root where `runs/` should be written,
- optional OpenAI environment variables if AI research planning is enabled.

Hermeneia does not yet include hosted auth, multi-user isolation, or publishing integrations, so expose the HTTP API only on trusted networks for this MVP release.
