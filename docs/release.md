# Release Process

This document defines the lightweight release process for Hermeneia.

## Versioning

Hermeneia uses semantic versioning:

- `v0.x.y` for pre-1.0 MVP and technical-preview releases.
- Patch releases for bug fixes that do not change command or API behavior.
- Minor releases for new CLI/API capabilities, templates, or workflow phases.

The first MVP release is `v0.1.0`.

## Pre-Release Checklist

Run these checks from a clean `main` checkout:

```bash
git pull
go test ./...
git diff --check
go build -o dist/hermeneia ./cmd/hermeneia
```

Confirm the working tree is clean:

```bash
git status --short
```

## Smoke Test

Use an isolated database:

```bash
SMOKE_DIR="$(mktemp -d)"
export HERMENEIA_DATABASE_PATH="$SMOKE_DIR/hermeneia.db"
./dist/hermeneia init
```

Create and render a carousel run:

```bash
./dist/hermeneia create --topic "AI agents in marketing" --type carousel
./dist/hermeneia list
./dist/hermeneia revise <run-id> --instruction "Make the hook sharper"
./dist/hermeneia render <run-id>
./dist/hermeneia show <run-id>
```

Create and render a short-video run:

```bash
./dist/hermeneia create --topic "AI agents in marketing" --type short_video
./dist/hermeneia render <run-id>
```

Check that the run folder contains:

- `brief.v1.json`
- `history.md`
- `content.json`
- carousel PNGs or `output/video/ai-news-short.mp4`

## API Smoke Test

Start the local API:

```bash
./dist/hermeneia serve
```

In another terminal:

```bash
curl -sS http://127.0.0.1:19318/health
curl -sS http://127.0.0.1:19318/v1/runs
curl -sS -X POST http://127.0.0.1:19318/v1/runs \
  -H "Content-Type: application/json" \
  --data '{"topic":"Release smoke","content_type":"carousel"}'
```

Expected health response:

```json
{"status":"ok"}
```

## Tagging

Only tag after tests and smoke checks pass:

```bash
git tag -a v0.1.0 -m "Hermeneia v0.1.0"
git push origin v0.1.0
```

## GitHub Release

Create the GitHub release from the tag:

```bash
gh release create v0.1.0 \
  --title "Hermeneia v0.1.0" \
  --notes-file CHANGELOG.md
```

For now, the release is source-first. Binary artifacts can be added later once cross-platform build automation is available.

## Post-Release

After publishing:

- Verify the release page exists.
- Verify `go install github.com/wauputr4/hermeneia/cmd/hermeneia@v0.1.0` works in a fresh environment.
- Open follow-up issues for any release notes marked as deferred.
