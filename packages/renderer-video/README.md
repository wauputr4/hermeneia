# Hermeneia Video Renderer

This package contains the first Remotion composition scaffold for the MVP video template:

- composition id: `AINewsShort`
- template id: `video/ai-news-short`
- input contract: `VideoContent` from `src/schema.ts`
- frame size: `1080x1920`
- aspect ratio: `9:16`

The Go CLI currently writes `runs/{run-id}/output/video/remotion-input.json` and uses a deterministic local MP4 renderer for the first MVP loop. The Remotion composition consumes the same structured scene contract so the TypeScript renderer can replace the temporary local renderer without changing workflow or storage code.
