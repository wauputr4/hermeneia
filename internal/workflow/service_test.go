package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/templates"
)

func TestServiceCreateReviseAndRenderCarouselRun(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Now = func() time.Time { return time.Date(2026, 5, 9, 7, 0, 0, 0, time.UTC) }
	ids := 0
	service.NewID = func(prefix, seed string) string {
		if prefix == "run" {
			return "run-ai-agents"
		}
		ids++
		return prefix + "-" + seed + "-" + string(rune('a'+ids))
	}

	created, err := service.CreateRun(ctx, CreateInput{Topic: "AI agents in marketing", ContentType: "carousel"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Run.ID != "run-ai-agents" {
		t.Fatalf("unexpected run id %q", created.Run.ID)
	}
	if _, err := os.Stat(created.BriefPath); err != nil {
		t.Fatal(err)
	}

	revised, err := service.ReviseRun(ctx, created.Run.ID, "Make the hook sharper")
	if err != nil {
		t.Fatal(err)
	}
	if revised.Brief.Version != 2 {
		t.Fatalf("expected v2, got v%d", revised.Brief.Version)
	}
	if _, err := os.Stat(revised.BriefPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(service.Files.BriefPath(created.Run.ID, 1)); err != nil {
		t.Fatalf("previous brief should be preserved: %v", err)
	}

	rendered, err := service.RenderRun(ctx, created.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered.Artifacts) == 0 {
		t.Fatal("expected render artifacts")
	}
	for _, artifact := range rendered.Artifacts {
		if artifact.Checksum == "" {
			t.Fatalf("artifact missing checksum: %#v", artifact)
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatal(err)
		}
	}
	scheduledAt := time.Date(2026, 5, 10, 2, 0, 0, 0, time.UTC)
	scheduled, err := service.SchedulePost(ctx, ScheduleInput{
		RunID:       created.Run.ID,
		ArtifactID:  rendered.Artifacts[0].ID,
		Platform:    "instagram",
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scheduled.Post.Status != "scheduled" || scheduled.Post.Platform != "instagram" {
		t.Fatalf("unexpected scheduled post: %#v", scheduled.Post)
	}
	if !strings.Contains(scheduled.Post.ValidationJSON, `"credentials_stored_in_db":false`) {
		t.Fatalf("validation must not store credentials: %s", scheduled.Post.ValidationJSON)
	}
	_, err = service.SchedulePost(ctx, ScheduleInput{
		RunID:       created.Run.ID,
		Platform:    "instagram",
		ScheduledAt: time.Date(2026, 5, 9, 6, 59, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected past schedule timestamp to be rejected")
	}
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "future") {
		t.Fatalf("unexpected schedule error: %v", err)
	}

	details, err := service.ShowRun(ctx, created.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Briefs) != 2 || len(details.Revisions) != 1 || len(details.Artifacts) == 0 || len(details.Schedules) != 1 {
		t.Fatalf("unexpected run details: %#v", details)
	}
	history, err := os.ReadFile(filepath.Join(service.Files.RunDir(created.Run.ID), "history.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(history), "Make the hook sharper") {
		t.Fatalf("history missing revision instruction:\n%s", history)
	}
	if !strings.Contains(string(history), "scheduled instagram post") {
		t.Fatalf("history missing schedule entry:\n%s", history)
	}
}

func TestServiceCreateRunFromResearchStoresTraceablePlan(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.NewID = func(prefix, seed string) string {
		switch prefix {
		case "run":
			return "run-research-ai"
		case "artifact":
			return "artifact-research-json"
		default:
			return prefix + "-test"
		}
	}

	result, err := service.CreateRunFromResearch(ctx, ResearchInput{
		Topic:       "AI agents in marketing",
		ContentType: "carousel",
		Sources: []ResearchSource{
			{URL: "https://example.com/agents", Note: "Agent adoption signal"},
			{URL: "https://example.com/marketing", Title: "Marketing workflows"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResearchArtifact.Kind != "research_json" || result.ResearchArtifact.Checksum == "" {
		t.Fatalf("unexpected research artifact: %#v", result.ResearchArtifact)
	}
	data, err := os.ReadFile(result.ResearchPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan ResearchPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Sources) != 2 || plan.Sources[0].URL != "https://example.com/agents" {
		t.Fatalf("source URLs were not preserved: %#v", plan.Sources)
	}
	brief, err := service.Repo.GetLatestBriefVersion(ctx, result.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(brief.BodyJSON, "source-backed") {
		t.Fatalf("brief was not generated from research plan: %s", brief.BodyJSON)
	}
}

func TestServiceCreateRunFromResearchDefaultsToDeterministicPlanner(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Planner = OpenAIResearchPlanner{APIKey: "test-key", Model: "test-model", BaseURL: "http://127.0.0.1:1"}
	service.NewID = func(prefix, seed string) string {
		switch prefix {
		case "run":
			return "run-default-deterministic-research"
		case "artifact":
			return "artifact-default-deterministic-research-json"
		default:
			return prefix + "-test"
		}
	}

	result, err := service.CreateRunFromResearch(ctx, ResearchInput{
		Topic:       "AI agents in marketing",
		ContentType: "carousel",
		Sources:     []ResearchSource{{URL: "https://example.com/agents"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.ResearchPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan ResearchPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Planner != "deterministic" {
		t.Fatalf("expected deterministic planner by default, got %q", plan.Planner)
	}
}

func TestServiceCreateRunFromResearchUsesOpenAIPlanner(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	var sawAuth bool
	openai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		sawAuth = r.Header.Get("Authorization") == "Bearer test-key"
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"output":[{"content":[{"type":"output_text","text":"{\"topic\":\"ignored\",\"sources\":[],\"summary\":\"AI planning signal from supplied sources.\",\"ideas\":[{\"title\":\"Agent workflow adoption\",\"reason\":\"The source note points to practical AI agent usage.\",\"rank\":1}],\"content_type\":\"carousel\",\"template_id\":\"carousel/ai-news-clean\",\"planner\":\"openai\"}"}]}]}`)
	}))
	defer openai.Close()

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Planner = OpenAIResearchPlanner{APIKey: "test-key", Model: "test-model", BaseURL: openai.URL, HTTPClient: openai.Client()}
	service.NewID = func(prefix, seed string) string {
		switch prefix {
		case "run":
			return "run-openai-research"
		case "artifact":
			return "artifact-openai-research-json"
		default:
			return prefix + "-test"
		}
	}

	result, err := service.CreateRunFromResearch(ctx, ResearchInput{
		Topic:       "AI agents in marketing",
		ContentType: "carousel",
		Planner:     "openai",
		Sources:     []ResearchSource{{URL: "https://example.com/agents", Note: "Agent workflow adoption"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth {
		t.Fatal("expected OpenAI authorization header")
	}
	data, err := os.ReadFile(result.ResearchPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan ResearchPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Planner != "openai" || len(plan.Sources) != 1 || plan.Sources[0].URL != "https://example.com/agents" {
		t.Fatalf("unexpected OpenAI plan: %#v", plan)
	}
	brief, err := service.Repo.GetLatestBriefVersion(ctx, result.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(brief.BodyJSON, "Agent workflow adoption") {
		t.Fatalf("brief did not use OpenAI research idea: %s", brief.BodyJSON)
	}
}

func TestServiceCreateRunFromResearchCleansUpOnArtifactFailure(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewRepository(db)
	if err := repo.EnsureTemplate(ctx, storage.Template{ID: "carousel/ai-news-clean", Name: "AI News Clean Carousel", ContentType: "carousel"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateContentRun(ctx, storage.ContentRun{ID: "existing-run", Topic: "Existing", ContentType: "carousel", TemplateID: "carousel/ai-news-clean"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateBriefVersion(ctx, storage.BriefVersion{ID: "existing-brief", RunID: "existing-run", Version: 1, BodyJSON: `{"topic":"Existing"}`}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateArtifact(ctx, storage.Artifact{ID: "artifact-research-json", RunID: "existing-run", BriefVersionID: "existing-brief", Kind: "research_json", Path: "runs/existing-run/research.json"}); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, runfiles.New(t.TempDir()))
	service.NewID = func(prefix, seed string) string {
		switch prefix {
		case "run":
			return "run-research-cleanup"
		case "artifact":
			return "artifact-research-json"
		default:
			return prefix + "-test"
		}
	}

	_, err = service.CreateRunFromResearch(ctx, ResearchInput{
		Topic:       "AI agents in marketing",
		ContentType: "carousel",
		Sources:     []ResearchSource{{URL: "https://example.com/agents"}},
	})
	if err == nil {
		t.Fatal("expected artifact insert error")
	}
	if _, err := repo.GetContentRun(ctx, "run-research-cleanup"); err != sql.ErrNoRows {
		t.Fatalf("expected content run cleanup, got %v", err)
	}
	if _, err := os.Stat(service.Files.RunDir("run-research-cleanup")); !os.IsNotExist(err) {
		t.Fatalf("expected runfiles cleanup, got %v", err)
	}
}

func TestServiceCreateRunPreservesExistingRunOnIDCollision(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewRepository(db)
	if err := repo.EnsureTemplate(ctx, storage.Template{ID: "carousel/ai-news-clean", Name: "AI News Clean Carousel", ContentType: "carousel"}); err != nil {
		t.Fatal(err)
	}
	existing := storage.ContentRun{ID: "run-collision", Topic: "Existing topic", ContentType: "carousel", TemplateID: "carousel/ai-news-clean"}
	if err := repo.CreateContentRun(ctx, existing); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, runfiles.New(t.TempDir()))
	service.NewID = func(prefix, seed string) string {
		if prefix == "run" {
			return "run-collision"
		}
		return prefix + "-test"
	}

	_, err = service.CreateRun(ctx, CreateInput{Topic: "AI agents in marketing", ContentType: "carousel"})
	if err == nil {
		t.Fatal("expected duplicate run id error")
	}
	run, err := repo.GetContentRun(ctx, "run-collision")
	if err != nil {
		t.Fatal(err)
	}
	if run.Topic != existing.Topic {
		t.Fatalf("existing run was changed: %#v", run)
	}
	if _, err := os.Stat(service.Files.RunDir("run-collision")); !os.IsNotExist(err) {
		t.Fatalf("expected collision runfiles cleanup, got %v", err)
	}
}

func TestServiceCleanupUsesUncancelledContext(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewRepository(db)
	if err := repo.EnsureTemplate(ctx, storage.Template{ID: "carousel/ai-news-clean", Name: "AI News Clean Carousel", ContentType: "carousel"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateContentRun(ctx, storage.ContentRun{ID: "run-cancel-cleanup", Topic: "AI agents", ContentType: "carousel", TemplateID: "carousel/ai-news-clean"}); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo, runfiles.New(t.TempDir()))
	if err := service.Files.PrepareRun("run-cancel-cleanup"); err != nil {
		t.Fatal(err)
	}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	service.cleanupPreparedRun(canceledCtx, "run-cancel-cleanup", true)

	if _, err := repo.GetContentRun(ctx, "run-cancel-cleanup"); err != sql.ErrNoRows {
		t.Fatalf("expected canceled-context cleanup to delete content run, got %v", err)
	}
	if _, err := os.Stat(service.Files.RunDir("run-cancel-cleanup")); !os.IsNotExist(err) {
		t.Fatalf("expected canceled-context cleanup to remove runfiles, got %v", err)
	}
}

func TestServiceListsAndValidatesTemplates(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	manifests, err := service.ListTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 2 {
		t.Fatalf("expected built-in templates, got %#v", manifests)
	}
	template, err := service.GetTemplate(ctx, "carousel/ai-news-clean")
	if err != nil {
		t.Fatal(err)
	}
	if template.Path == "" || template.ContentType != ContentTypeCarousel {
		t.Fatalf("unexpected template: %#v", template)
	}
	_, err = service.CreateRun(ctx, CreateInput{Topic: "AI agents", ContentType: "carousel", TemplateID: "missing/template"})
	if err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "template not found") {
		t.Fatalf("expected unknown template validation error, got %v", err)
	}
	_, err = service.CreateRun(ctx, CreateInput{Topic: "AI agents", ContentType: "carousel", TemplateID: "video/ai-news-short"})
	if err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "not \"carousel\"") {
		t.Fatalf("expected incompatible template validation error, got %v", err)
	}
}

func TestServiceCreateRunValidatesTemplateRequiredInput(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	writeWorkflowTestManifest(t, root, "carousel/strict", `{
  "id": "carousel/strict",
  "name": "Strict Carousel",
  "content_type": "carousel",
  "description": "Requires a field the default carousel content does not provide.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["content_json", "carousel_png"],
  "input_schema": {"type":"object","required":["template","missing_field"]}
}`)
	catalog, err := templates.LoadRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Templates = catalog
	_, err = service.CreateRun(ctx, CreateInput{Topic: "AI agents", ContentType: "carousel", TemplateID: "carousel/strict"})
	if err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "$.missing_field is required") {
		t.Fatalf("expected template input validation error, got %v", err)
	}
}

func TestServiceCreateRunValidatesVideoTemplateInput(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	_, err = service.CreateRun(ctx, CreateInput{Topic: "AI agents", ContentType: "short_video", TemplateID: "video/ai-news-short"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceRenderRunValidatesTemplateLimitsBeforeWritingArtifacts(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	writeWorkflowTestManifest(t, root, "carousel/limited", limitedCarouselManifest("carousel/limited", 10))
	catalog, err := templates.LoadRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Templates = catalog
	service.NewID = func(prefix, seed string) string {
		if prefix == "run" {
			return "run-limited"
		}
		return prefix + "-limited"
	}
	created, err := service.CreateRun(ctx, CreateInput{Topic: "AI agents", ContentType: "carousel", TemplateID: "carousel/limited"})
	if err != nil {
		t.Fatal(err)
	}

	writeWorkflowTestManifest(t, root, "carousel/limited", limitedCarouselManifest("carousel/limited", 1))
	catalog, err = templates.LoadRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	service.Templates = catalog

	_, err = service.RenderRun(ctx, created.Run.ID)
	if err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "$.slides must contain at most 1") {
		t.Fatalf("expected render input validation error, got %v", err)
	}
	if _, statErr := os.Stat(service.Files.ContentPath(created.Run.ID)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("content should not be written before validation, stat err=%v", statErr)
	}
	artifacts, listErr := service.Repo.ListArtifactsByRun(ctx, created.Run.ID)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected no artifacts after validation failure, got %#v", artifacts)
	}
}

func TestServiceShowRunRequiresExistingRun(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	if _, err := service.ShowRun(ctx, "missing"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func writeWorkflowTestManifest(t *testing.T, root, id, body string) {
	t.Helper()
	fullDir := filepath.Join(root, filepath.FromSlash(id))
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullDir, "template.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func limitedCarouselManifest(id string, maxSlides int) string {
	return fmt.Sprintf(`{
  "id": %q,
  "name": "Limited Carousel",
  "content_type": "carousel",
  "description": "A carousel manifest with a slide limit.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["content_json", "carousel_png"],
  "input_schema": {
    "type": "object",
    "required": ["template", "slides", "caption", "hashtags"],
    "properties": {
      "template": {"const": %q},
      "slides": {"type": "array", "minItems": 1, "maxItems": %d},
      "caption": {"type": "string"},
      "hashtags": {"type": "array"}
    }
  }
}`, id, id, maxSlides)
}
