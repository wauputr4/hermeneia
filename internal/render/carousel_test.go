package render

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCarouselRendererWritesPNGSlidesAndCaption(t *testing.T) {
	content := CarouselContent{
		Template: TemplateCarouselAINewsClean,
		Slides: []CarouselSlide{
			{Type: "cover", Headline: "AI agents are changing marketing", Body: "A practical overview"},
			{Type: "point", Headline: "Why it matters", Body: "Teams need repeatable workflows."},
		},
		Caption: "Caption draft",
	}
	outputDir := t.TempDir()
	files, err := CarouselRenderer{}.Render(context.Background(), content, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	firstSlide, err := os.Open(filepath.Join(outputDir, "slide-01.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer firstSlide.Close()
	cfg, err := png.DecodeConfig(firstSlide)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 1080 || cfg.Height != 1350 {
		t.Fatalf("unexpected PNG dimensions %dx%d", cfg.Width, cfg.Height)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "caption.txt")); err != nil {
		t.Fatal(err)
	}
}
