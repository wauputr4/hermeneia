package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/workflow"
)

const maxRequestBodyBytes = 1 << 20

type Server struct {
	service workflow.Service
	mux     *http.ServeMux
}

func New(service workflow.Service) *Server {
	server := &Server{service: service, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if applyLocalCORS(w, r) && r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/runs", s.handleListRuns)
	s.mux.HandleFunc("POST /v1/runs", s.handleCreateRun)
	s.mux.HandleFunc("POST /v1/research-runs", s.handleCreateResearchRun)
	s.mux.HandleFunc("GET /v1/runs/{runID}", s.handleShowRun)
	s.mux.HandleFunc("DELETE /v1/runs/{runID}", s.handleDeleteRun)
	s.mux.HandleFunc("GET /v1/runs/{runID}/briefs", s.handleListBriefs)
	s.mux.HandleFunc("GET /v1/runs/{runID}/artifacts", s.handleListArtifacts)
	s.mux.HandleFunc("POST /v1/runs/{runID}/revisions", s.handleReviseRun)
	s.mux.HandleFunc("POST /v1/runs/{runID}/render", s.handleRenderRun)
	s.mux.HandleFunc("POST /v1/runs/{runID}/schedule", s.handleSchedulePost)
	s.mux.HandleFunc("GET /v1/scheduled-posts", s.handleListScheduledPosts)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.service.ListRuns(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]runResponse, 0, len(runs))
	for _, run := range runs {
		out = append(out, newRunResponse(run))
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": out})
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := s.service.CreateRun(r.Context(), req.toInput())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, createRunResponse{
		Run:         newRunResponse(result.Run),
		Brief:       newBriefResponse(result.Brief),
		BriefPath:   result.BriefPath,
		HistoryPath: result.HistoryPath,
	})
}

func (s *Server) handleCreateResearchRun(w http.ResponseWriter, r *http.Request) {
	var req createResearchRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := s.service.CreateRunFromResearch(r.Context(), req.toInput())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, createResearchRunResponse{
		createRunResponse: createRunResponse{
			Run:         newRunResponse(result.Run),
			Brief:       newBriefResponse(result.Brief),
			BriefPath:   result.BriefPath,
			HistoryPath: result.HistoryPath,
		},
		ResearchPath:     result.ResearchPath,
		ResearchArtifact: newArtifactResponse(result.ResearchArtifact),
	})
}

func (s *Server) handleShowRun(w http.ResponseWriter, r *http.Request) {
	details, err := s.service.ShowRun(r.Context(), r.PathValue("runID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newRunDetailsResponse(details))
}

func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	if err := s.service.DeleteRun(r.Context(), r.PathValue("runID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListBriefs(w http.ResponseWriter, r *http.Request) {
	briefVersions, err := s.service.ListBriefs(r.Context(), r.PathValue("runID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	briefs := make([]briefResponse, 0, len(briefVersions))
	for _, brief := range briefVersions {
		briefs = append(briefs, newBriefResponse(brief))
	}
	writeJSON(w, http.StatusOK, map[string]any{"briefs": briefs})
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	artifactRows, err := s.service.ListArtifacts(r.Context(), r.PathValue("runID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	artifacts := make([]artifactResponse, 0, len(artifactRows))
	for _, artifact := range artifactRows {
		artifacts = append(artifacts, newArtifactResponse(artifact))
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": artifacts})
}

func (s *Server) handleReviseRun(w http.ResponseWriter, r *http.Request) {
	var req reviseRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := s.service.ReviseRun(r.Context(), r.PathValue("runID"), req.Instruction)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, reviseRunResponse{
		Run:       newRunResponse(result.Run),
		Previous:  newBriefResponse(result.Previous),
		Brief:     newBriefResponse(result.Brief),
		BriefPath: result.BriefPath,
	})
}

func (s *Server) handleRenderRun(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.RenderRun(r.Context(), r.PathValue("runID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	artifacts := make([]artifactResponse, 0, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts = append(artifacts, newArtifactResponse(artifact))
	}
	writeJSON(w, http.StatusCreated, renderRunResponse{
		Run:       newRunResponse(result.Run),
		Brief:     newBriefResponse(result.Brief),
		Content:   result.Content,
		Artifacts: artifacts,
	})
}

func (s *Server) handleSchedulePost(w http.ResponseWriter, r *http.Request) {
	var req schedulePostRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := s.service.SchedulePost(r.Context(), req.toInput(r.PathValue("runID")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, schedulePostResponse{
		Run:  newRunResponse(result.Run),
		Post: newScheduledPostResponse(result.Post),
	})
}

func (s *Server) handleListScheduledPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := s.service.ListScheduledPosts(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]scheduledPostResponse, 0, len(posts))
	for _, post := range posts {
		out = append(out, newScheduledPostResponse(post))
	}
	writeJSON(w, http.StatusOK, map[string]any{"scheduled_posts": out})
}

type createRunRequest struct {
	Topic          string `json:"topic"`
	ContentType    string `json:"content_type"`
	TemplateID     string `json:"template_id"`
	Tone           string `json:"tone"`
	Platform       string `json:"platform"`
	TargetAudience string `json:"target_audience"`
}

func (r createRunRequest) toInput() workflow.CreateInput {
	return workflow.CreateInput{
		Topic:          r.Topic,
		ContentType:    r.ContentType,
		TemplateID:     r.TemplateID,
		Tone:           r.Tone,
		Platform:       r.Platform,
		TargetAudience: r.TargetAudience,
	}
}

type createResearchRunRequest struct {
	createRunRequest
	Sources []workflow.ResearchSource `json:"sources"`
	Planner string                    `json:"planner"`
}

func (r createResearchRunRequest) toInput() workflow.ResearchInput {
	input := r.createRunRequest.toInput()
	return workflow.ResearchInput{
		Topic:          input.Topic,
		ContentType:    input.ContentType,
		TemplateID:     input.TemplateID,
		Tone:           input.Tone,
		Platform:       input.Platform,
		TargetAudience: input.TargetAudience,
		Sources:        r.Sources,
		Planner:        r.Planner,
	}
}

type reviseRunRequest struct {
	Instruction string `json:"instruction"`
}

type schedulePostRequest struct {
	ArtifactID  string    `json:"artifact_id"`
	Platform    string    `json:"platform"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func (r schedulePostRequest) toInput(runID string) workflow.ScheduleInput {
	return workflow.ScheduleInput{
		RunID:       runID,
		ArtifactID:  r.ArtifactID,
		Platform:    r.Platform,
		ScheduledAt: r.ScheduledAt,
	}
}

type createRunResponse struct {
	Run         runResponse   `json:"run"`
	Brief       briefResponse `json:"brief"`
	BriefPath   string        `json:"brief_path"`
	HistoryPath string        `json:"history_path"`
}

type createResearchRunResponse struct {
	createRunResponse
	ResearchPath     string           `json:"research_path"`
	ResearchArtifact artifactResponse `json:"research_artifact"`
}

type reviseRunResponse struct {
	Run       runResponse   `json:"run"`
	Previous  briefResponse `json:"previous"`
	Brief     briefResponse `json:"brief"`
	BriefPath string        `json:"brief_path"`
}

type renderRunResponse struct {
	Run       runResponse        `json:"run"`
	Brief     briefResponse      `json:"brief"`
	Content   any                `json:"content"`
	Artifacts []artifactResponse `json:"artifacts"`
}

type schedulePostResponse struct {
	Run  runResponse           `json:"run"`
	Post scheduledPostResponse `json:"scheduled_post"`
}

type runDetailsResponse struct {
	Run       runResponse             `json:"run"`
	Briefs    []briefResponse         `json:"briefs"`
	Revisions []revisionResponse      `json:"revisions"`
	Artifacts []artifactResponse      `json:"artifacts"`
	Schedules []scheduledPostResponse `json:"scheduled_posts"`
}

func newRunDetailsResponse(details workflow.RunDetails) runDetailsResponse {
	briefs := make([]briefResponse, 0, len(details.Briefs))
	for _, brief := range details.Briefs {
		briefs = append(briefs, newBriefResponse(brief))
	}
	revisions := make([]revisionResponse, 0, len(details.Revisions))
	for _, revision := range details.Revisions {
		revisions = append(revisions, newRevisionResponse(revision))
	}
	artifacts := make([]artifactResponse, 0, len(details.Artifacts))
	for _, artifact := range details.Artifacts {
		artifacts = append(artifacts, newArtifactResponse(artifact))
	}
	schedules := make([]scheduledPostResponse, 0, len(details.Schedules))
	for _, schedule := range details.Schedules {
		schedules = append(schedules, newScheduledPostResponse(schedule))
	}
	return runDetailsResponse{
		Run:       newRunResponse(details.Run),
		Briefs:    briefs,
		Revisions: revisions,
		Artifacts: artifacts,
		Schedules: schedules,
	}
}

type runResponse struct {
	ID          string    `json:"id"`
	Topic       string    `json:"topic"`
	ContentType string    `json:"content_type"`
	TemplateID  string    `json:"template_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func newRunResponse(run storage.ContentRun) runResponse {
	return runResponse{
		ID:          run.ID,
		Topic:       run.Topic,
		ContentType: run.ContentType,
		TemplateID:  run.TemplateID,
		CreatedAt:   run.CreatedAt,
	}
}

type briefResponse struct {
	ID        string          `json:"id"`
	RunID     string          `json:"run_id"`
	Version   int             `json:"version"`
	Body      json.RawMessage `json:"body"`
	CreatedAt time.Time       `json:"created_at"`
}

func newBriefResponse(brief storage.BriefVersion) briefResponse {
	return briefResponse{
		ID:        brief.ID,
		RunID:     brief.RunID,
		Version:   brief.Version,
		Body:      json.RawMessage(brief.BodyJSON),
		CreatedAt: brief.CreatedAt,
	}
}

type artifactResponse struct {
	ID             string    `json:"id"`
	RunID          string    `json:"run_id"`
	BriefVersionID string    `json:"brief_version_id,omitempty"`
	Kind           string    `json:"kind"`
	Path           string    `json:"path"`
	Checksum       string    `json:"checksum,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func newArtifactResponse(artifact storage.Artifact) artifactResponse {
	return artifactResponse{
		ID:             artifact.ID,
		RunID:          artifact.RunID,
		BriefVersionID: artifact.BriefVersionID,
		Kind:           artifact.Kind,
		Path:           artifact.Path,
		Checksum:       artifact.Checksum,
		CreatedAt:      artifact.CreatedAt,
	}
}

type revisionResponse struct {
	ID                 string    `json:"id"`
	RunID              string    `json:"run_id"`
	FromBriefVersionID string    `json:"from_brief_version_id"`
	ToBriefVersionID   string    `json:"to_brief_version_id"`
	Instruction        string    `json:"instruction"`
	CreatedAt          time.Time `json:"created_at"`
}

func newRevisionResponse(revision storage.RevisionEvent) revisionResponse {
	return revisionResponse{
		ID:                 revision.ID,
		RunID:              revision.RunID,
		FromBriefVersionID: revision.FromBriefVersionID,
		ToBriefVersionID:   revision.ToBriefVersionID,
		Instruction:        revision.Instruction,
		CreatedAt:          revision.CreatedAt,
	}
}

type scheduledPostResponse struct {
	ID          string          `json:"id"`
	RunID       string          `json:"run_id"`
	ArtifactID  string          `json:"artifact_id,omitempty"`
	Platform    string          `json:"platform"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	Status      string          `json:"status"`
	Validation  json.RawMessage `json:"validation"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func newScheduledPostResponse(post storage.ScheduledPost) scheduledPostResponse {
	validation := json.RawMessage(`{}`)
	if post.ValidationJSON != "" {
		validation = json.RawMessage(post.ValidationJSON)
	}
	return scheduledPostResponse{
		ID:          post.ID,
		RunID:       post.RunID,
		ArtifactID:  post.ArtifactID,
		Platform:    post.Platform,
		ScheduledAt: post.ScheduledAt,
		Status:      post.Status,
		Validation:  validation,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, errors.New("request body must contain a single JSON object"))
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, workflow.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("{\"error\":\"encode response\"}\n"))
		return
	}
	data = append(data, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func applyLocalCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || !isLoopbackOrigin(origin) {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	return true
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return parsed.Scheme == "http" && (host == "127.0.0.1" || host == "localhost" || host == "::1")
}
