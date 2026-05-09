package render

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVideoRendererWritesRemotionInputAndMP4(t *testing.T) {
	outputDir := t.TempDir()
	renderer := VideoRenderer{
		FFmpegPath: "/usr/bin/ffmpeg",
		RunCommand: func(ctx context.Context, name string, args ...string) error {
			if name != "/usr/bin/ffmpeg" {
				t.Fatalf("unexpected command %q", name)
			}
			if len(args) == 0 {
				t.Fatal("expected ffmpeg args")
			}
			return os.WriteFile(args[len(args)-1], []byte("fake mp4"), 0o644)
		},
	}

	files, err := renderer.Render(context.Background(), VideoContent{
		Template:    TemplateVideoAINewsShort,
		AspectRatio: "9:16",
		FPS:         30,
		Scenes: []VideoScene{
			{DurationSeconds: 3, Text: "Hook", Visual: "Clean layout"},
		},
		Caption: "Caption",
	}, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	input, err := os.ReadFile(filepath.Join(outputDir, "remotion-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(input), `"aspect_ratio": "9:16"`) {
		t.Fatalf("remotion input missing aspect ratio:\n%s", input)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "ai-news-short.mp4")); err != nil {
		t.Fatal(err)
	}
}
