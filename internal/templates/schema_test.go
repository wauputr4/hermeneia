package templates

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateInputAcceptsBuiltInCarouselContract(t *testing.T) {
	catalog, err := LoadDir(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := catalog.Get("carousel/ai-news-clean")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateInput(manifest, map[string]any{
		"template": "carousel/ai-news-clean",
		"slides": []map[string]any{
			{"type": "cover", "headline": "Hook", "body": "Angle"},
		},
		"caption":  "Caption",
		"hashtags": []string{"#Hermeneia"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateInputRejectsMissingRequiredField(t *testing.T) {
	manifest := Manifest{
		ID:          "carousel/strict",
		InputSchema: []byte(`{"type":"object","required":["template","slides"]}`),
	}

	err := ValidateInput(manifest, map[string]any{"template": "carousel/strict"})
	if err == nil || !strings.Contains(err.Error(), "$.slides is required") {
		t.Fatalf("expected missing required field error, got %v", err)
	}
}

func TestValidateInputRejectsArrayLimit(t *testing.T) {
	manifest := Manifest{
		ID:          "video/short",
		InputSchema: []byte(`{"type":"object","properties":{"scenes":{"type":"array","maxItems":1}}}`),
	}

	err := ValidateInput(manifest, map[string]any{
		"scenes": []map[string]any{
			{"text": "one"},
			{"text": "two"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "$.scenes must contain at most 1") {
		t.Fatalf("expected maxItems error, got %v", err)
	}
}

func TestValidateInputAcceptsNullConst(t *testing.T) {
	manifest := Manifest{
		ID:          "carousel/null-const",
		InputSchema: []byte(`{"type":"object","properties":{"optional":{"const":null}}}`),
	}

	err := ValidateInput(manifest, map[string]any{"optional": nil})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateInputFormatsEnumValuesAsJSON(t *testing.T) {
	manifest := Manifest{
		ID:          "carousel/enum",
		InputSchema: []byte(`{"type":"object","properties":{"slide":{"enum":[{"type":"cover"},"closing"]}}}`),
	}

	err := ValidateInput(manifest, map[string]any{"slide": map[string]any{"type": "point"}})
	if err == nil || !strings.Contains(err.Error(), `$.slide must be one of {"type":"cover"}, "closing"`) {
		t.Fatalf("expected JSON-formatted enum error, got %v", err)
	}
}
