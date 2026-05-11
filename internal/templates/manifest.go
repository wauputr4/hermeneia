package templates

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
	Root         string          `json:"-"`
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

func LoadConfigured() (Catalog, error) {
	root, err := findBuiltInRoot()
	if err != nil {
		return Catalog{}, err
	}
	roots := []string{root}
	roots = append(roots, TemplateRootsFromEnv()...)
	return LoadRoots(roots)
}

func TemplateRootsFromEnv() []string {
	value := strings.TrimSpace(os.Getenv("HERMENEIA_TEMPLATE_PATH"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, string(os.PathListSeparator))
	roots := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			roots = append(roots, part)
		}
	}
	return roots
}

func LoadDir(root string) (Catalog, error) {
	return LoadRoots([]string{root})
}

func LoadRoots(roots []string) (Catalog, error) {
	var paths []string
	pathRoots := make(map[string]string)
	seenRoots := make(map[string]struct{})
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "." || root == "" {
			continue
		}
		if _, ok := seenRoots[root]; ok {
			continue
		}
		seenRoots[root] = struct{}{}
		rootPaths, err := manifestPaths(root)
		if err != nil {
			return Catalog{}, err
		}
		if len(rootPaths) == 0 {
			return Catalog{}, fmt.Errorf("no template manifests found in %s", root)
		}
		sort.Strings(rootPaths)
		for _, path := range rootPaths {
			pathRoots[filepath.Clean(path)] = root
		}
		paths = append(paths, rootPaths...)
	}

	catalog := Catalog{
		byID:     make(map[string]Manifest),
		defaults: make(map[string]string),
	}
	var pathErr error
	for _, path := range paths {
		manifest, err := loadManifest(path)
		if err != nil {
			return Catalog{}, err
		}
		manifest.Root = pathRoots[filepath.Clean(path)]
		if err := validateManifest(manifest); err != nil {
			return Catalog{}, err
		}
		if existing, exists := catalog.byID[manifest.ID]; exists {
			return Catalog{}, fmt.Errorf("duplicate template id %q in %s conflicts with %s", manifest.ID, manifest.Root, existing.Root)
		}
		if err := validateManifestPath(manifest.Root, manifest); err != nil && pathErr == nil {
			pathErr = err
		}
		catalog.byID[manifest.ID] = manifest
		catalog.items = append(catalog.items, manifest)
		if _, exists := catalog.defaults[manifest.ContentType]; !exists {
			catalog.defaults[manifest.ContentType] = manifest.ID
		}
	}
	if pathErr != nil {
		return Catalog{}, pathErr
	}
	if len(catalog.items) == 0 {
		return Catalog{}, errors.New("no template manifests found")
	}
	return catalog, nil
}

func manifestPaths(root string) ([]string, error) {
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
		return nil, err
	}
	return paths, nil
}

func (c Catalog) All() []Manifest {
	out := make([]Manifest, len(c.items))
	copy(out, c.items)
	return out
}

func (c Catalog) Len() int {
	return len(c.items)
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
	dec := json.NewDecoder(bytes.NewReader(data))
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
	if err := validateAssetPath(manifest.Path, "preview_asset", manifest.PreviewAsset); err != nil {
		return err
	}
	for _, asset := range manifest.Assets {
		if err := validateAssetPath(manifest.Path, "assets", asset); err != nil {
			return err
		}
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

func validateAssetPath(manifestPath, field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("%s: %s path %q must stay inside the template directory", manifestPath, field, value)
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
		if info, err := os.Stat(candidate); err == nil && info.IsDir() && hasTemplateManifest(candidate) {
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

func hasTemplateManifest(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if !d.IsDir() && filepath.Base(path) == "template.json" {
			found = true
		}
		return nil
	})
	return found
}
