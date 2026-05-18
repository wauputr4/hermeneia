package httpapi

import (
	"bytes"
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

	"github.com/wauputr4/hermeneia/internal/render"
	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/templates"
	"github.com/wauputr4/hermeneia/internal/workflow"
)

func TestServerContentRunWorkflow(t *testing.T) {
	handler := newTestHandler(t)

	create := request(t, handler, http.MethodPost, "/v1/runs", `{"topic":"AI agents in marketing","content_type":"carousel"}`)
	assertStatus(t, create, http.StatusCreated)
	var created createRunResponse
	decodeResponse(t, create, &created)
	if created.Run.ID == "" || created.Brief.Version != 1 {
		t.Fatalf("unexpected create response: %#v", created)
	}
	if created.Run.CreatedAt.IsZero() || created.Brief.CreatedAt.IsZero() {
		t.Fatalf("expected create response timestamps, got run=%s brief=%s", created.Run.CreatedAt, created.Brief.CreatedAt)
	}

	list := request(t, handler, http.MethodGet, "/v1/runs", "")
	assertStatus(t, list, http.StatusOK)
	var listed struct {
		Runs []runResponse `json:"runs"`
	}
	decodeResponse(t, list, &listed)
	if len(listed.Runs) != 1 || listed.Runs[0].ID != created.Run.ID {
		t.Fatalf("unexpected run list: %#v", listed.Runs)
	}

	show := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID, "")
	assertStatus(t, show, http.StatusOK)
	var details runDetailsResponse
	decodeResponse(t, show, &details)
	if len(details.Briefs) != 1 {
		t.Fatalf("expected one brief, got %#v", details.Briefs)
	}

	revise := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/revisions", `{"instruction":"Make the hook sharper"}`)
	assertStatus(t, revise, http.StatusCreated)
	var revised reviseRunResponse
	decodeResponse(t, revise, &revised)
	if revised.Brief.Version != 2 || revised.Previous.Version != 1 {
		t.Fatalf("unexpected revision response: %#v", revised)
	}
	if revised.Brief.CreatedAt.IsZero() {
		t.Fatalf("expected revision timestamp, got %s", revised.Brief.CreatedAt)
	}

	briefs := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/briefs", "")
	assertStatus(t, briefs, http.StatusOK)
	var briefList struct {
		Briefs []briefResponse `json:"briefs"`
	}
	decodeResponse(t, briefs, &briefList)
	if len(briefList.Briefs) != 2 {
		t.Fatalf("expected two briefs, got %#v", briefList.Briefs)
	}

	rendered := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/render", "")
	assertStatus(t, rendered, http.StatusCreated)
	var renderResult renderRunResponse
	decodeResponse(t, rendered, &renderResult)
	if len(renderResult.Artifacts) == 0 {
		t.Fatalf("expected render artifacts: %#v", renderResult)
	}
	if renderResult.Artifacts[0].CreatedAt.IsZero() {
		t.Fatalf("expected render artifact timestamp, got %s", renderResult.Artifacts[0].CreatedAt)
	}
	scheduledAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	schedule := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/schedule", `{"platform":"instagram","artifact_id":"`+renderResult.Artifacts[0].ID+`","scheduled_at":"`+scheduledAt+`"}`)
	assertStatus(t, schedule, http.StatusCreated)
	var scheduled schedulePostResponse
	decodeResponse(t, schedule, &scheduled)
	if scheduled.Post.Status != "scheduled" || !strings.Contains(string(scheduled.Post.Validation), `"credentials_stored_in_db":false`) {
		t.Fatalf("unexpected schedule response: %#v", scheduled)
	}
	youtubeScheduledAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	youtubeSchedule := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/schedule", `{"platform":"youtube","scheduled_at":"`+youtubeScheduledAt+`"}`)
	assertStatus(t, youtubeSchedule, http.StatusCreated)
	var youtubeScheduled schedulePostResponse
	decodeResponse(t, youtubeSchedule, &youtubeScheduled)

	artifacts := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifacts", "")
	assertStatus(t, artifacts, http.StatusOK)
	var artifactList struct {
		Artifacts []artifactResponse `json:"artifacts"`
	}
	decodeResponse(t, artifacts, &artifactList)
	if len(artifactList.Artifacts) != len(renderResult.Artifacts) {
		t.Fatalf("artifact list mismatch: %#v", artifactList.Artifacts)
	}
	schedules := request(t, handler, http.MethodGet, "/v1/scheduled-posts", "")
	assertStatus(t, schedules, http.StatusOK)
	var scheduleList struct {
		ScheduledPosts []scheduledPostResponse `json:"scheduled_posts"`
	}
	decodeResponse(t, schedules, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 2 || scheduleList.ScheduledPosts[0].ID != scheduled.Post.ID || scheduleList.ScheduledPosts[1].ID != youtubeScheduled.Post.ID {
		t.Fatalf("schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	instagramSchedules := request(t, handler, http.MethodGet, "/v1/scheduled-posts?platform=instagram", "")
	assertStatus(t, instagramSchedules, http.StatusOK)
	decodeResponse(t, instagramSchedules, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 1 || scheduleList.ScheduledPosts[0].Platform != "instagram" {
		t.Fatalf("platform-filtered schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	scheduledSchedules := request(t, handler, http.MethodGet, "/v1/scheduled-posts?status=scheduled", "")
	assertStatus(t, scheduledSchedules, http.StatusOK)
	decodeResponse(t, scheduledSchedules, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 2 || scheduleList.ScheduledPosts[0].Status != "scheduled" || scheduleList.ScheduledPosts[1].Status != "scheduled" {
		t.Fatalf("status-filtered schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	scheduledInstagram := request(t, handler, http.MethodGet, "/v1/scheduled-posts?status=scheduled&platform=instagram", "")
	assertStatus(t, scheduledInstagram, http.StatusOK)
	decodeResponse(t, scheduledInstagram, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 1 || scheduleList.ScheduledPosts[0].ID != scheduled.Post.ID {
		t.Fatalf("combined-filter schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	rangeFiltered := request(t, handler, http.MethodGet, "/v1/scheduled-posts?from="+scheduledAt+"&to="+scheduledAt, "")
	assertStatus(t, rangeFiltered, http.StatusOK)
	decodeResponse(t, rangeFiltered, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 1 || scheduleList.ScheduledPosts[0].ID != scheduled.Post.ID {
		t.Fatalf("range-filtered schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	combinedRange := request(t, handler, http.MethodGet, "/v1/scheduled-posts?status=scheduled&platform=youtube&from="+scheduledAt+"&to="+youtubeScheduledAt, "")
	assertStatus(t, combinedRange, http.StatusOK)
	decodeResponse(t, combinedRange, &scheduleList)
	if len(scheduleList.ScheduledPosts) != 1 || scheduleList.ScheduledPosts[0].ID != youtubeScheduled.Post.ID {
		t.Fatalf("combined range-filter schedule list mismatch: %#v", scheduleList.ScheduledPosts)
	}
	invalidFilter := request(t, handler, http.MethodGet, "/v1/scheduled-posts?status=queued", "")
	assertStatus(t, invalidFilter, http.StatusBadRequest)
	invalidFrom := request(t, handler, http.MethodGet, "/v1/scheduled-posts?from=tomorrow", "")
	assertStatus(t, invalidFrom, http.StatusBadRequest)
	invertedRange := request(t, handler, http.MethodGet, "/v1/scheduled-posts?from="+youtubeScheduledAt+"&to="+scheduledAt, "")
	assertStatus(t, invertedRange, http.StatusBadRequest)
	cancelled := request(t, handler, http.MethodPatch, "/v1/scheduled-posts/"+scheduled.Post.ID, `{"status":"cancelled"}`)
	assertStatus(t, cancelled, http.StatusOK)
	var cancelledPost schedulePostResponse
	decodeResponse(t, cancelled, &cancelledPost)
	if cancelledPost.Post.Status != "cancelled" {
		t.Fatalf("unexpected cancelled schedule response: %#v", cancelledPost)
	}
	invalidStatus := request(t, handler, http.MethodPatch, "/v1/scheduled-posts/"+scheduled.Post.ID, `{"status":"published"}`)
	assertStatus(t, invalidStatus, http.StatusBadRequest)
	var previewArtifact artifactResponse
	for _, artifact := range renderResult.Artifacts {
		if artifact.Kind == "carousel_png" {
			previewArtifact = artifact
			break
		}
	}
	if previewArtifact.ID == "" {
		t.Fatalf("expected previewable carousel artifact: %#v", renderResult.Artifacts)
	}
	file := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifacts/"+previewArtifact.ID+"/file", "")
	assertStatus(t, file, http.StatusOK)
	if file.Body.String() != "fake png" {
		t.Fatalf("unexpected artifact file body: %q", file.Body.String())
	}

	deleted := request(t, handler, http.MethodDelete, "/v1/runs/"+created.Run.ID, "")
	assertStatus(t, deleted, http.StatusNoContent)
	missing := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID, "")
	assertStatus(t, missing, http.StatusNotFound)
}

func TestServerRejectsArtifactFileSymlinkEscape(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)
	runID := "run-symlink"
	briefID := "brief-symlink"
	if err := service.Repo.CreateContentRun(ctx, storage.ContentRun{ID: runID, Topic: "AI agents", ContentType: "carousel"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Repo.CreateBriefVersion(ctx, storage.BriefVersion{ID: briefID, RunID: runID, Version: 1, BodyJSON: `{"topic":"AI agents"}`}); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(service.Files.RunDir(runID), "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outsidePath, []byte("outside secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(outputDir, "slide-01.png")
	if err := os.Symlink(outsidePath, artifactPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := service.Repo.CreateArtifact(ctx, storage.Artifact{
		ID:             "artifact-symlink",
		RunID:          runID,
		BriefVersionID: briefID,
		Kind:           "carousel_png",
		Path:           artifactPath,
	}); err != nil {
		t.Fatal(err)
	}

	file := request(t, New(&service), http.MethodGet, "/v1/runs/"+runID+"/artifacts/artifact-symlink/file", "")

	assertStatus(t, file, http.StatusBadRequest)
}

func TestServerResearchRunAndValidation(t *testing.T) {
	handler := newTestHandler(t)

	research := request(t, handler, http.MethodPost, "/v1/research-runs", `{
		"topic":"AI agents in marketing",
		"content_type":"carousel",
		"sources":[{"url":"https://example.com/agents","title":"Agent workflows"}]
	}`)
	assertStatus(t, research, http.StatusCreated)
	var created createResearchRunResponse
	decodeResponse(t, research, &created)
	if created.ResearchPath == "" || created.ResearchArtifact.Kind != "research_json" {
		t.Fatalf("unexpected research response: %#v", created)
	}
	if created.Run.CreatedAt.IsZero() || created.Brief.CreatedAt.IsZero() || created.ResearchArtifact.CreatedAt.IsZero() {
		t.Fatalf("expected research response timestamps: %#v", created)
	}

	invalid := request(t, handler, http.MethodPost, "/v1/runs", `{"content_type":"carousel"}`)
	assertStatus(t, invalid, http.StatusBadRequest)

	unknownField := request(t, handler, http.MethodPost, "/v1/runs", `{"topic":"AI agents","unexpected":true}`)
	assertStatus(t, unknownField, http.StatusBadRequest)

	missing := request(t, handler, http.MethodGet, "/v1/runs/missing", "")
	assertStatus(t, missing, http.StatusNotFound)
}

func TestServerCreateRunFromWorkflowPreset(t *testing.T) {
	handler := newTestHandler(t)

	create := request(t, handler, http.MethodPost, "/v1/runs", `{"workflow_id":"simple-carousel","topic":"AI agents in marketing"}`)
	assertStatus(t, create, http.StatusCreated)
	var created workflowRunResponse
	decodeResponse(t, create, &created)
	if created.Run.ID == "" || created.Run.ContentType != "carousel" || created.Run.TemplateID != "carousel/ai-news-clean" {
		t.Fatalf("unexpected workflow run response: %#v", created)
	}
	if len(created.Artifacts) == 0 {
		t.Fatalf("expected rendered workflow artifacts: %#v", created)
	}

	missingInput := request(t, handler, http.MethodPost, "/v1/runs", `{"workflow_id":"research-carousel","topic":"AI agents"}`)
	assertStatus(t, missingInput, http.StatusBadRequest)
	unknown := request(t, handler, http.MethodPost, "/v1/runs", `{"workflow_id":"missing","topic":"AI agents"}`)
	assertStatus(t, unknown, http.StatusNotFound)
}

func TestServerArtifactAuditPassesForHealthyRender(t *testing.T) {
	handler := newTestHandler(t)

	create := request(t, handler, http.MethodPost, "/v1/runs", `{"topic":"AI agents in marketing","content_type":"carousel"}`)
	assertStatus(t, create, http.StatusCreated)
	var created createRunResponse
	decodeResponse(t, create, &created)

	rendered := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/render", "")
	assertStatus(t, rendered, http.StatusCreated)

	audit := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifact-audit", "")
	assertStatus(t, audit, http.StatusOK)
	var result artifactAuditResponse
	decodeResponse(t, audit, &result)
	if !result.Healthy || len(result.Issues) != 0 || result.Run.ID != created.Run.ID {
		t.Fatalf("unexpected healthy audit response: %#v", result)
	}
}

func TestServerArtifactAuditReturnsDriftPayload(t *testing.T) {
	service := newTestService(t)
	handler := New(&service)

	create := request(t, handler, http.MethodPost, "/v1/runs", `{"topic":"AI agents in marketing","content_type":"carousel"}`)
	assertStatus(t, create, http.StatusCreated)
	var created createRunResponse
	decodeResponse(t, create, &created)

	rendered := request(t, handler, http.MethodPost, "/v1/runs/"+created.Run.ID+"/render", "")
	assertStatus(t, rendered, http.StatusCreated)
	var renderResult renderRunResponse
	decodeResponse(t, rendered, &renderResult)
	if len(renderResult.Artifacts) == 0 {
		t.Fatal("expected rendered artifacts")
	}
	if err := os.Remove(renderResult.Artifacts[0].Path); err != nil {
		t.Fatal(err)
	}

	audit := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifact-audit", "")
	assertStatus(t, audit, http.StatusConflict)
	var result artifactAuditResponse
	decodeResponse(t, audit, &result)
	if result.Healthy || result.Run.ID != created.Run.ID {
		t.Fatalf("unexpected drift audit response: %#v", result)
	}
	if !artifactAuditResponseContains(result.Issues, "missing_file") {
		t.Fatalf("expected missing file issue, got %#v", result.Issues)
	}

	file := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifacts/"+renderResult.Artifacts[0].ID+"/file", "")
	assertStatus(t, file, http.StatusNotFound)
}

func TestServerTemplateCatalog(t *testing.T) {
	handler := newTestHandler(t)

	list := request(t, handler, http.MethodGet, "/v1/templates", "")
	assertStatus(t, list, http.StatusOK)
	var listed struct {
		Templates []templateResponse `json:"templates"`
	}
	decodeResponse(t, list, &listed)
	if len(listed.Templates) != 2 {
		t.Fatalf("expected built-in templates, got %#v", listed.Templates)
	}
	if listed.Templates[0].ID == "" || listed.Templates[0].InputSchema == nil {
		t.Fatalf("unexpected template list response: %#v", listed.Templates)
	}

	show := request(t, handler, http.MethodGet, "/v1/templates/carousel/ai-news-clean", "")
	assertStatus(t, show, http.StatusOK)
	var detail struct {
		Template templateResponse `json:"template"`
	}
	decodeResponse(t, show, &detail)
	if detail.Template.ID != "carousel/ai-news-clean" || detail.Template.Renderer == "" {
		t.Fatalf("unexpected template detail: %#v", detail.Template)
	}

	missing := request(t, handler, http.MethodGet, "/v1/templates/missing/template", "")
	assertStatus(t, missing, http.StatusNotFound)
}

func TestServerTemplateCatalogIncludesCustomRoots(t *testing.T) {
	customRoot := t.TempDir()
	writeHTTPTestManifest(t, customRoot, "carousel/custom-local")
	catalog, err := templates.LoadRoots([]string{filepath.Join("..", "..", "templates"), customRoot})
	if err != nil {
		t.Fatal(err)
	}
	service := newTestService(t)
	service.Templates = catalog
	handler := New(&service)

	list := request(t, handler, http.MethodGet, "/v1/templates", "")
	assertStatus(t, list, http.StatusOK)
	var listed struct {
		Templates []templateResponse `json:"templates"`
	}
	decodeResponse(t, list, &listed)
	var found bool
	for _, template := range listed.Templates {
		if template.ID == "carousel/custom-local" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected custom template in API response: %#v", listed.Templates)
	}
}

func TestServerWorkflowCatalog(t *testing.T) {
	handler := newTestHandler(t)

	list := request(t, handler, http.MethodGet, "/v1/workflows", "")
	assertStatus(t, list, http.StatusOK)
	var listed struct {
		Workflows []workflowPresetResponse `json:"workflows"`
	}
	decodeResponse(t, list, &listed)
	if len(listed.Workflows) != 2 {
		t.Fatalf("expected built-in workflows, got %#v", listed.Workflows)
	}
	if listed.Workflows[0].ID == "" || len(listed.Workflows[0].Steps) == 0 || len(listed.Workflows[0].RequiredInputs) == 0 {
		t.Fatalf("unexpected workflow list response: %#v", listed.Workflows)
	}

	show := request(t, handler, http.MethodGet, "/v1/workflows/research-carousel", "")
	assertStatus(t, show, http.StatusOK)
	var detail struct {
		Workflow workflowPresetResponse `json:"workflow"`
	}
	decodeResponse(t, show, &detail)
	if detail.Workflow.ID != "research-carousel" || detail.Workflow.DefaultTemplateID != "carousel/ai-news-clean" || len(detail.Workflow.Steps) != 3 {
		t.Fatalf("unexpected workflow detail: %#v", detail.Workflow)
	}

	missing := request(t, handler, http.MethodGet, "/v1/workflows/missing", "")
	assertStatus(t, missing, http.StatusNotFound)
}

func TestWriteServiceErrorStatusMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: sql.ErrNoRows, want: http.StatusNotFound},
		{name: "validation", err: fmt.Errorf("%w: topic is required", workflow.ErrInvalidInput), want: http.StatusBadRequest},
		{name: "unexpected", err: errors.New("database is locked"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeServiceError(rec, tt.err)
			assertStatus(t, rec, tt.want)
		})
	}
}

func TestWriteJSONEncodesBeforeCommittingStatus(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusOK, map[string]any{"bad": make(chan int)})

	assertStatus(t, rec, http.StatusInternalServerError)
	if rec.Body.String() != "{\"error\":\"encode response\"}\n" {
		t.Fatalf("unexpected encode error body: %q", rec.Body.String())
	}
}

func TestServerAllowsLoopbackCORSPreflight(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodOptions, "/v1/runs", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func TestServerRejectsNonLoopbackCORSOrigin(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	service := newTestService(t)
	return New(&service)
}

func newTestService(t *testing.T) workflow.Service {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "hermeneia.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := storage.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	service := workflow.NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	ids := 0
	service.NewID = func(prefix, seed string) string {
		ids++
		if prefix == "run" {
			return fmt.Sprintf("run-%d", ids)
		}
		return fmt.Sprintf("%s-%d", prefix, ids)
	}
	service.Carousel = fakeCarouselRenderer{}
	return service
}

type fakeCarouselRenderer struct{}

func (fakeCarouselRenderer) Render(ctx context.Context, content render.CarouselContent, outputDir string) ([]render.OutputFile, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(outputDir, "slide-01.png")
	if err := os.WriteFile(path, []byte("fake png"), 0o644); err != nil {
		return nil, err
	}
	return []render.OutputFile{{Kind: "carousel_png", Path: path}}, nil
}

func writeHTTPTestManifest(t *testing.T, root, id string) {
	t.Helper()
	fullDir := filepath.Join(root, filepath.FromSlash(id))
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "id": "` + id + `",
  "name": "Custom Local",
  "content_type": "carousel",
  "description": "A custom local template.",
  "version": "1.0.0",
  "aspect_ratio": "4:5",
  "renderer": "go-png",
  "output_kinds": ["carousel_png"],
  "input_schema": {}
}`
	if err := os.WriteFile(filepath.Join(fullDir, "template.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func request(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected status %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
}

func artifactAuditResponseContains(issues []artifactAuditIssueResponse, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rec.Body.String())
	}
}
