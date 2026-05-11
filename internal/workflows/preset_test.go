package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wauputr4/hermeneia/internal/templates"
)

func TestLoadDirLoadsBuiltInPresets(t *testing.T) {
	templateCatalog, err := templates.LoadDir(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := LoadDir(filepath.Join("..", "..", "workflows"), templateCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.All()); got != 2 {
		t.Fatalf("expected 2 workflow presets, got %d", got)
	}
	preset, err := catalog.Get("simple-carousel")
	if err != nil {
		t.Fatal(err)
	}
	if preset.DefaultTemplateID != "carousel/ai-news-clean" || len(preset.Steps) != 2 {
		t.Fatalf("unexpected preset: %#v", preset)
	}
}

func TestLoadFilesRejectsDuplicatePresetIDs(t *testing.T) {
	templateCatalog := testTemplateCatalog(t)
	first := writePreset(t, validPreset("simple-carousel", "carousel/clean", "carousel"))
	second := writePreset(t, validPreset("simple-carousel", "carousel/clean", "carousel"))

	_, err := LoadFiles([]string{first, second}, templateCatalog)
	if err == nil || !strings.Contains(err.Error(), "duplicate workflow preset id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestValidatePresetRejectsUnknownStepType(t *testing.T) {
	templateCatalog := testTemplateCatalog(t)
	preset := Preset{
		Path:              "preset.json",
		ID:                "unknown-step",
		Name:              "Unknown Step",
		Description:       "Uses an unsupported step.",
		ContentType:       "carousel",
		DefaultTemplateID: "carousel/clean",
		Steps:             []Step{{Type: "execute_script"}},
	}

	err := ValidatePreset(preset, templateCatalog)
	if err == nil || !strings.Contains(err.Error(), `unsupported type "execute_script"`) {
		t.Fatalf("expected unsupported step error, got %v", err)
	}
}

func TestValidatePresetRejectsMissingTemplateReference(t *testing.T) {
	templateCatalog := testTemplateCatalog(t)
	preset := Preset{
		Path:              "preset.json",
		ID:                "missing-template",
		Name:              "Missing Template",
		Description:       "References a template that is not installed.",
		ContentType:       "carousel",
		DefaultTemplateID: "carousel/missing",
		Steps:             []Step{{Type: StepCreateBrief}},
	}

	err := ValidatePreset(preset, templateCatalog)
	if err == nil || !strings.Contains(err.Error(), `default_template_id "carousel/missing" not found`) {
		t.Fatalf("expected missing template error, got %v", err)
	}
}

func TestValidatePresetRejectsInvalidContentType(t *testing.T) {
	templateCatalog := testTemplateCatalog(t)
	preset := Preset{
		Path:              "preset.json",
		ID:                "invalid-content",
		Name:              "Invalid Content",
		Description:       "Uses an unsupported content type.",
		ContentType:       "thread",
		DefaultTemplateID: "carousel/clean",
		Steps:             []Step{{Type: StepCreateBrief}},
	}

	err := ValidatePreset(preset, templateCatalog)
	if err == nil || !strings.Contains(err.Error(), `unsupported content_type "thread"`) {
		t.Fatalf("expected invalid content type error, got %v", err)
	}
}

func TestValidatePresetRejectsTemplateContentMismatch(t *testing.T) {
	templateCatalog := testTemplateCatalog(t)
	preset := Preset{
		Path:              "preset.json",
		ID:                "mismatch",
		Name:              "Mismatch",
		Description:       "Uses a video template for carousel content.",
		ContentType:       "carousel",
		DefaultTemplateID: "video/short",
		Steps:             []Step{{Type: StepCreateBrief}},
	}

	err := ValidatePreset(preset, templateCatalog)
	if err == nil || !strings.Contains(err.Error(), `not "carousel"`) {
		t.Fatalf("expected content mismatch error, got %v", err)
	}
}

func testTemplateCatalog(t *testing.T) templates.Catalog {
	t.Helper()
	root := t.TempDir()
	writeTemplateManifest(t, root, "carousel/clean", "carousel")
	writeTemplateManifest(t, root, "video/short", "short_video")
	catalog, err := templates.LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func writeTemplateManifest(t *testing.T, root, id, contentType string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(id))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "id": "` + id + `",
  "name": "Test Template",
  "content_type": "` + contentType + `",
  "description": "Template for workflow preset tests.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "test",
  "output_kinds": ["content_json"],
  "input_schema": {"type":"object"}
}`
	if err := os.WriteFile(filepath.Join(dir, "template.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePreset(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "preset.json")
	if err := os.WriteFile(path, []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func validPreset(id, templateID, contentType string) string {
	return `{
  "id": "` + id + `",
  "name": "Simple Carousel",
  "description": "Create and render a carousel.",
  "content_type": "` + contentType + `",
  "default_template_id": "` + templateID + `",
  "steps": [{"type":"create_brief"}, {"type":"render"}],
  "required_inputs": ["topic"]
}`
}
