package templates

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ContentTypeCarousel   = "carousel"
	ContentTypeShortVideo = "short_video"
)

var ErrNotFound = errors.New("template not found")

type Manifest struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	ContentType  string          `json:"content_type"`
	Description  string          `json:"description"`
	Version      string          `json:"version"`
	AspectRatio  string          `json:"aspect_ratio"`
	Renderer     string          `json:"renderer"`
	OutputKinds  []string        `json:"output_kinds"`
	InputSchema  json.RawMessage `json:"input_schema"`
	PreviewAsset string          `json:"preview_asset,omitempty"`
	Assets       []string        `json:"assets,omitempty"`
	Path         string          `json:"-"`
}

type Catalog struct {
	items    []Manifest
	byID     map[string]Manifest
	defaults map[string]string
}

func LoadBuiltIn() (Catalog, error) {
	root, err := findBuiltInRoot()
	if err != nil {
		return Catalog{}, err
	}
	return LoadDir(root)
}

func LoadDir(root string) (Catalog, error) {
	root = filepath.Clean(root)
	var paths []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "template.json" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return Catalog{}, err
	}
	sort.Strings(paths)

	catalog := Catalog{
		byID:     make(map[string]Manifest),
		defaults: make(map[string]string),
	}
	for _, path := range paths {
		manifest, err := loadManifest(path)
		if err != nil {
			return Catalog{}, err
		}
		if err := validateManifest(manifest); err != nil {
			return Catalog{}, err
		}
		if _, exists := catalog.byID[manifest.ID]; exists {
			return Catalog{}, fmt.Errorf("duplicate template id %q", manifest.ID)
		}
		catalog.byID[manifest.ID] = manifest
		catalog.items = append(catalog.items, manifest)
		if _, exists := catalog.defaults[manifest.ContentType]; !exists {
			catalog.defaults[manifest.ContentType] = manifest.ID
		}
	}
	for _, manifest := range catalog.items {
		if err := validateManifestPath(root, manifest); err != nil {
			return Catalog{}, err
		}
	}
	if len(catalog.items) == 0 {
		return Catalog{}, fmt.Errorf("no template manifests found in %s", root)
	}
	return catalog, nil
}

func (c Catalog) All() []Manifest {
	out := make([]Manifest, len(c.items))
	copy(out, c.items)
	return out
}

func (c Catalog) Get(id string) (Manifest, error) {
	id = strings.TrimSpace(id)
	manifest, ok := c.byID[id]
	if !ok {
		return Manifest{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return manifest, nil
}

func (c Catalog) Default(contentType string) (Manifest, error) {
	contentType = strings.TrimSpace(contentType)
	id, ok := c.defaults[contentType]
	if !ok {
		return Manifest{}, fmt.Errorf("%w for content type %q", ErrNotFound, contentType)
	}
	return c.byID[id], nil
}

func loadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", path, err)
	}
	manifest.Path = filepath.Clean(path)
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	required := map[string]string{
		"id":           manifest.ID,
		"name":         manifest.Name,
		"content_type": manifest.ContentType,
		"description":  manifest.Description,
		"version":      manifest.Version,
		"aspect_ratio": manifest.AspectRatio,
		"renderer":     manifest.Renderer,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: %s is required", manifest.Path, field)
		}
	}
	if len(manifest.OutputKinds) == 0 {
		return fmt.Errorf("%s: output_kinds is required", manifest.Path)
	}
	if len(manifest.InputSchema) == 0 || !json.Valid(manifest.InputSchema) {
		return fmt.Errorf("%s: input_schema must be valid JSON", manifest.Path)
	}
	if !supportedContentType(manifest.ContentType) {
		return fmt.Errorf("%s: unsupported content_type %q", manifest.Path, manifest.ContentType)
	}
	return nil
}

func validateManifestPath(root string, manifest Manifest) error {
	wantPath := filepath.Join(root, filepath.FromSlash(manifest.ID), "template.json")
	if filepath.Clean(manifest.Path) != filepath.Clean(wantPath) {
		return fmt.Errorf("%s: id %q must map to %s", manifest.Path, manifest.ID, wantPath)
	}
	return nil
}

func supportedContentType(contentType string) bool {
	switch contentType {
	case ContentTypeCarousel, ContentTypeShortVideo:
		return true
	default:
		return false
	}
}

func findBuiltInRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "templates")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() && fileExists(filepath.Join(dir, "go.mod")) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("built-in templates directory not found")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
