package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDirLoadsBuiltInManifests(t *testing.T) {
	catalog, err := LoadDir(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.All()); got != 2 {
		t.Fatalf("expected 2 built-in manifests, got %d", got)
	}
	carousel, err := catalog.Get("carousel/ai-news-clean")
	if err != nil {
		t.Fatal(err)
	}
	if carousel.Name != "AI News Clean Carousel" || carousel.ContentType != ContentTypeCarousel {
		t.Fatalf("unexpected carousel manifest: %#v", carousel)
	}
	video, err := catalog.Default(ContentTypeShortVideo)
	if err != nil {
		t.Fatal(err)
	}
	if video.ID != "video/ai-news-short" {
		t.Fatalf("unexpected video default: %#v", video)
	}
}

func TestLoadDirRejectsMissingRequiredField(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "carousel/broken", `{
  "id": "carousel/broken",
  "content_type": "carousel",
  "description": "Broken manifest",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["carousel_png"],
  "input_schema": {}
}`)

	_, err := LoadDir(root)
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestLoadDirRejectsDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "carousel/one", validManifest("carousel/duplicate", "carousel"))
	writeManifest(t, root, "carousel/two", validManifest("carousel/duplicate", "carousel"))

	_, err := LoadDir(root)
	if err == nil || !strings.Contains(err.Error(), "duplicate template id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestLoadDirRejectsUnsupportedContentType(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "thread/ai-news", validManifest("thread/ai-news", "thread"))

	_, err := LoadDir(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported content_type") {
		t.Fatalf("expected unsupported content type error, got %v", err)
	}
}

func TestLoadDirRejectsIDPathMismatch(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "carousel/wrong-path", validManifest("carousel/right-path", "carousel"))

	_, err := LoadDir(root)
	if err == nil || !strings.Contains(err.Error(), "must map to") {
		t.Fatalf("expected id path mismatch error, got %v", err)
	}
}

func writeManifest(t *testing.T, root, dir, body string) {
	t.Helper()
	fullDir := filepath.Join(root, filepath.FromSlash(dir))
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullDir, "template.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validManifest(id, contentType string) string {
	return `{
  "id": "` + id + `",
  "name": "Valid Template",
  "content_type": "` + contentType + `",
  "description": "A valid template manifest.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["carousel_png"],
  "input_schema": {}
}`
}
