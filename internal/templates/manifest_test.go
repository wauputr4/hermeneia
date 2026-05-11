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
	if got := catalog.Len(); got != 2 {
		t.Fatalf("expected 2 built-in manifests, got %d", got)
	}
	if got := len(catalog.All()); got != catalog.Len() {
		t.Fatalf("expected All to return every manifest, got %d", got)
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

func TestLoadBuiltInDoesNotRequireGoMod(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "templates"), "carousel/portable", validManifest("carousel/portable", "carousel"))

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(root, "templates", "carousel", "portable")); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	catalog, err := LoadBuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Len(); got != 1 {
		t.Fatalf("expected 1 manifest without go.mod, got %d", got)
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

func TestLoadRootsMergesMultipleTemplateRoots(t *testing.T) {
	builtIn := t.TempDir()
	custom := t.TempDir()
	writeManifest(t, builtIn, "carousel/built-in", validManifest("carousel/built-in", "carousel"))
	writeManifest(t, custom, "video/custom-short", validManifest("video/custom-short", "short_video"))

	catalog, err := LoadRoots([]string{builtIn, custom})
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Len(); got != 2 {
		t.Fatalf("expected 2 manifests, got %d", got)
	}
	if _, err := catalog.Get("carousel/built-in"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get("video/custom-short"); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRootsRejectsDuplicateIDsAcrossRoots(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	writeManifest(t, first, "carousel/duplicate", validManifest("carousel/duplicate", "carousel"))
	writeManifest(t, second, "carousel/duplicate", validManifest("carousel/duplicate", "carousel"))

	_, err := LoadRoots([]string{first, second})
	if err == nil || !strings.Contains(err.Error(), first) || !strings.Contains(err.Error(), second) {
		t.Fatalf("expected duplicate error naming both roots, got %v", err)
	}
}

func TestLoadRootsRejectsRootsWithoutManifests(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "carousel", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRoots([]string{root})
	if err == nil || !strings.Contains(err.Error(), "no template manifests found") {
		t.Fatalf("expected missing manifest error, got %v", err)
	}
}

func TestLoadRootsSkipsEmptyRoots(t *testing.T) {
	first := t.TempDir()
	empty := t.TempDir()
	if err := os.MkdirAll(filepath.Join(empty, "carousel", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, first, "carousel/valid", validManifest("carousel/valid", "carousel"))

	catalog, err := LoadRoots([]string{empty, first})
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Len(); got != 1 {
		t.Fatalf("expected one manifest after skipping empty root, got %d", got)
	}
	if _, err := catalog.Get("carousel/valid"); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfiguredIncludesEnvTemplatePath(t *testing.T) {
	root := t.TempDir()
	custom := t.TempDir()
	writeManifest(t, filepath.Join(root, "templates"), "carousel/built-in", validManifest("carousel/built-in", "carousel"))
	writeManifest(t, custom, "carousel/custom", validManifest("carousel/custom", "carousel"))

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	t.Setenv("HERMENEIA_TEMPLATE_PATH", custom)

	catalog, err := LoadConfigured()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get("carousel/built-in"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get("carousel/custom"); err != nil {
		t.Fatal(err)
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

func TestLoadDirRejectsAssetPathTraversal(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "carousel/traversal", `{
  "id": "carousel/traversal",
  "name": "Traversal",
  "content_type": "carousel",
  "description": "A manifest with unsafe asset paths.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["carousel_png"],
  "input_schema": {},
  "preview_asset": "../preview.png"
}`)

	_, err := LoadDir(root)
	if err == nil || !strings.Contains(err.Error(), "must stay inside the template directory") {
		t.Fatalf("expected path traversal error, got %v", err)
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
