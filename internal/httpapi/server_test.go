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
	"testing"

	"github.com/wauputr4/hermeneia/internal/render"
	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
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

	artifacts := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID+"/artifacts", "")
	assertStatus(t, artifacts, http.StatusOK)
	var artifactList struct {
		Artifacts []artifactResponse `json:"artifacts"`
	}
	decodeResponse(t, artifacts, &artifactList)
	if len(artifactList.Artifacts) != len(renderResult.Artifacts) {
		t.Fatalf("artifact list mismatch: %#v", artifactList.Artifacts)
	}

	deleted := request(t, handler, http.MethodDelete, "/v1/runs/"+created.Run.ID, "")
	assertStatus(t, deleted, http.StatusNoContent)
	missing := request(t, handler, http.MethodGet, "/v1/runs/"+created.Run.ID, "")
	assertStatus(t, missing, http.StatusNotFound)
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

	invalid := request(t, handler, http.MethodPost, "/v1/runs", `{"content_type":"carousel"}`)
	assertStatus(t, invalid, http.StatusBadRequest)

	unknownField := request(t, handler, http.MethodPost, "/v1/runs", `{"topic":"AI agents","unexpected":true}`)
	assertStatus(t, unknownField, http.StatusBadRequest)

	missing := request(t, handler, http.MethodGet, "/v1/runs/missing", "")
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

func newTestHandler(t *testing.T) http.Handler {
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
	return New(service)
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

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rec.Body.String())
	}
}
