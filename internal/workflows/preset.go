package workflows

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wauputr4/hermeneia/internal/templates"
)

const (
	StepCreateBrief    = "create_brief"
	StepResearchPlan   = "research_plan"
	StepReviseBrief    = "revise_brief"
	StepRender         = "render"
	StepScheduleRecord = "schedule_record"
)

var ErrNotFound = errors.New("workflow preset not found")

type Preset struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	ContentType       string         `json:"content_type"`
	DefaultTemplateID string         `json:"default_template_id"`
	Steps             []Step         `json:"steps"`
	RequiredInputs    []string       `json:"required_inputs"`
	RevisionPolicy    RevisionPolicy `json:"revision_policy,omitempty"`
	Path              string         `json:"-"`
}

type Step struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type RevisionPolicy struct {
	Mode         string `json:"mode,omitempty"`
	MaxRevisions int    `json:"max_revisions,omitempty"`
}

type Catalog struct {
	items []Preset
	byID  map[string]Preset
}

func LoadDir(root string, templateCatalog templates.Catalog) (Catalog, error) {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return Catalog{}, err
	}
	sort.Strings(paths)
	return LoadFiles(paths, templateCatalog)
}

func LoadFiles(paths []string, templateCatalog templates.Catalog) (Catalog, error) {
	catalog := Catalog{byID: make(map[string]Preset)}
	for _, path := range paths {
		preset, err := loadFile(path)
		if err != nil {
			return Catalog{}, err
		}
		if existing, ok := catalog.byID[preset.ID]; ok {
			return Catalog{}, fmt.Errorf("duplicate workflow preset id %q in %s conflicts with %s", preset.ID, preset.Path, existing.Path)
		}
		if err := ValidatePreset(preset, templateCatalog); err != nil {
			return Catalog{}, err
		}
		catalog.byID[preset.ID] = preset
		catalog.items = append(catalog.items, preset)
	}
	if len(catalog.items) == 0 {
		return Catalog{}, errors.New("no workflow presets found")
	}
	return catalog, nil
}

func ValidatePreset(preset Preset, templateCatalog templates.Catalog) error {
	required := map[string]string{
		"id":                  preset.ID,
		"name":                preset.Name,
		"description":         preset.Description,
		"content_type":        preset.ContentType,
		"default_template_id": preset.DefaultTemplateID,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: %s is required", preset.Path, field)
		}
	}
	if !supportedContentType(preset.ContentType) {
		return fmt.Errorf("%s: unsupported content_type %q", preset.Path, preset.ContentType)
	}
	manifest, err := templateCatalog.Get(preset.DefaultTemplateID)
	if err != nil {
		return fmt.Errorf("%s: default_template_id %q not found", preset.Path, preset.DefaultTemplateID)
	}
	if manifest.ContentType != preset.ContentType {
		return fmt.Errorf("%s: default_template_id %q is for content type %q, not %q", preset.Path, manifest.ID, manifest.ContentType, preset.ContentType)
	}
	if len(preset.Steps) == 0 {
		return fmt.Errorf("%s: steps is required", preset.Path)
	}
	for i, step := range preset.Steps {
		if !supportedStepType(step.Type) {
			return fmt.Errorf("%s: steps[%d] has unsupported type %q", preset.Path, i, step.Type)
		}
	}
	return nil
}

func (c Catalog) All() []Preset {
	out := make([]Preset, len(c.items))
	copy(out, c.items)
	return out
}

func (c Catalog) Get(id string) (Preset, error) {
	id = strings.TrimSpace(id)
	preset, ok := c.byID[id]
	if !ok {
		return Preset{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return preset, nil
}

func loadFile(path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, err
	}
	var preset Preset
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&preset); err != nil {
		return Preset{}, fmt.Errorf("%s: %w", path, err)
	}
	preset.Path = filepath.Clean(path)
	return preset, nil
}

func supportedContentType(contentType string) bool {
	switch contentType {
	case templates.ContentTypeCarousel, templates.ContentTypeShortVideo:
		return true
	default:
		return false
	}
}

func supportedStepType(stepType string) bool {
	switch stepType {
	case StepCreateBrief, StepResearchPlan, StepReviseBrief, StepRender, StepScheduleRecord:
		return true
	default:
		return false
	}
}
