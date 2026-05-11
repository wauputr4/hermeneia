package workflow

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/wauputr4/hermeneia/internal/brief"
	"github.com/wauputr4/hermeneia/internal/render"
	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/templates"
)

const (
	ContentTypeCarousel   = "carousel"
	ContentTypeShortVideo = "short_video"
)

var ErrInvalidInput = errors.New("invalid input")

type inputError string

func (e inputError) Error() string {
	return string(e)
}

func (e inputError) Unwrap() error {
	return ErrInvalidInput
}

func invalidInput(message string) error {
	return inputError(message)
}

type CarouselRenderer interface {
	Render(context.Context, render.CarouselContent, string) ([]render.OutputFile, error)
}

type VideoRenderer interface {
	Render(context.Context, render.VideoContent, string) ([]render.OutputFile, error)
}

type Service struct {
	Repo      *storage.Repository
	Files     runfiles.Store
	Carousel  CarouselRenderer
	Video     VideoRenderer
	Planner   ResearchPlanner
	Templates templates.Catalog
	Now       func() time.Time
	NewID     func(prefix, seed string) string
}

type CreateInput struct {
	Topic          string
	ContentType    string
	TemplateID     string
	Tone           string
	Platform       string
	TargetAudience string
}

type ResearchInput struct {
	Topic          string
	ContentType    string
	TemplateID     string
	Platform       string
	Tone           string
	TargetAudience string
	Sources        []ResearchSource
	Planner        string
}

type ResearchSource struct {
	URL   string `json:"url"`
	Note  string `json:"note,omitempty"`
	Title string `json:"title,omitempty"`
}

type ResearchPlan struct {
	Topic       string           `json:"topic"`
	Sources     []ResearchSource `json:"sources"`
	Summary     string           `json:"summary"`
	Ideas       []ResearchIdea   `json:"ideas"`
	ContentType string           `json:"content_type"`
	TemplateID  string           `json:"template_id"`
	Planner     string           `json:"planner"`
}

type ResearchIdea struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
	Rank   int    `json:"rank"`
}

type CreateResult struct {
	Run         storage.ContentRun
	Brief       storage.BriefVersion
	BriefPath   string
	HistoryPath string
}

type ResearchResult struct {
	CreateResult
	ResearchPath     string
	ResearchArtifact storage.Artifact
}

type ReviseResult struct {
	Run       storage.ContentRun
	Previous  storage.BriefVersion
	Brief     storage.BriefVersion
	BriefPath string
}

type RenderResult struct {
	Run       storage.ContentRun
	Brief     storage.BriefVersion
	Content   any
	Artifacts []storage.Artifact
}

type ScheduleInput struct {
	RunID       string
	ArtifactID  string
	Platform    string
	ScheduledAt time.Time
}

type ScheduleResult struct {
	Run  storage.ContentRun
	Post storage.ScheduledPost
}

type RunDetails struct {
	Run       storage.ContentRun
	Briefs    []storage.BriefVersion
	Revisions []storage.RevisionEvent
	Artifacts []storage.Artifact
	Schedules []storage.ScheduledPost
}

type ResearchPlanner interface {
	PlanResearch(context.Context, ResearchPlanningInput) (ResearchPlan, error)
}

type ResearchPlanningInput struct {
	Topic       string
	ContentType string
	TemplateID  string
	Sources     []ResearchSource
}

func NewService(repo *storage.Repository, files runfiles.Store) Service {
	service := Service{
		Repo:     repo,
		Files:    files,
		Carousel: render.CarouselRenderer{},
		Video:    render.VideoRenderer{},
		Planner:  DeterministicResearchPlanner{},
	}
	if catalog, err := templates.LoadConfigured(); err == nil {
		service.Templates = catalog
	}
	return service
}

func (s Service) CreateRun(ctx context.Context, input CreateInput) (CreateResult, error) {
	contentType, err := normalizeContentType(input.ContentType)
	if err != nil {
		return CreateResult{}, err
	}
	if strings.TrimSpace(input.Topic) == "" {
		return CreateResult{}, invalidInput("topic is required")
	}
	template, err := s.resolveTemplate(contentType, input.TemplateID)
	if err != nil {
		return CreateResult{}, err
	}
	templateID := template.ID

	b := draftBrief(input, contentType, templateID)
	return s.createRunWithBrief(ctx, input, contentType, template, b, fmt.Sprintf("- v1 created from topic %q.\n", b.Topic))
}

func (s Service) createRunWithBrief(ctx context.Context, input CreateInput, contentType string, template templates.Manifest, b brief.Brief, historyEntry string) (CreateResult, error) {
	runID := s.newID("run", input.Topic)
	templateID := template.ID
	if err := s.Files.PrepareRun(runID); err != nil {
		return CreateResult{}, err
	}
	runCreated := false
	fail := func(err error) (CreateResult, error) {
		s.cleanupPreparedRun(ctx, runID, runCreated)
		return CreateResult{}, err
	}

	body, err := marshalBrief(b)
	if err != nil {
		return fail(err)
	}

	if err := s.Repo.EnsureTemplate(ctx, storage.Template{ID: templateID, Name: template.Name, ContentType: contentType}); err != nil {
		return fail(err)
	}
	run := storage.ContentRun{ID: runID, Topic: b.Topic, ContentType: contentType, TemplateID: templateID}
	if err := s.Repo.CreateContentRun(ctx, run); err != nil {
		return fail(err)
	}
	runCreated = true
	storedRun, err := s.Repo.GetContentRun(ctx, runID)
	if err != nil {
		return fail(err)
	}
	version := storage.BriefVersion{ID: briefID(runID, 1), RunID: runID, Version: 1, BodyJSON: body}
	if err := s.Repo.CreateBriefVersion(ctx, version); err != nil {
		return fail(err)
	}
	storedVersion, err := s.Repo.GetBriefVersion(ctx, version.ID)
	if err != nil {
		return fail(err)
	}

	briefPath := s.Files.BriefPath(runID, 1)
	if err := runfiles.WriteJSON(briefPath, b); err != nil {
		return fail(err)
	}
	historyPath := s.Files.HistoryPath(runID)
	if err := runfiles.WriteText(historyPath, "# Hermeneia Run History\n\n"+historyEntry); err != nil {
		return fail(err)
	}
	return CreateResult{Run: storedRun, Brief: storedVersion, BriefPath: briefPath, HistoryPath: historyPath}, nil
}

func (s Service) cleanupCreatedRun(ctx context.Context, runID string) {
	s.cleanupPreparedRun(ctx, runID, true)
}

func (s Service) cleanupPreparedRun(ctx context.Context, runID string, deleteDB bool) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if deleteDB {
		_ = s.Repo.DeleteContentRun(cleanupCtx, runID)
	}
	_ = s.Files.RemoveRun(runID)
}

func (s Service) CreateRunFromResearch(ctx context.Context, input ResearchInput) (ResearchResult, error) {
	contentType, err := normalizeContentType(input.ContentType)
	if err != nil {
		return ResearchResult{}, err
	}
	if strings.TrimSpace(input.Topic) == "" {
		return ResearchResult{}, invalidInput("topic is required")
	}
	sources := normalizeResearchSources(input.Sources)
	if len(sources) == 0 {
		return ResearchResult{}, invalidInput("at least one source URL is required")
	}
	template, err := s.resolveTemplate(contentType, input.TemplateID)
	if err != nil {
		return ResearchResult{}, err
	}
	templateID := template.ID

	createInput := CreateInput{
		Topic:          input.Topic,
		ContentType:    contentType,
		TemplateID:     templateID,
		Tone:           input.Tone,
		Platform:       input.Platform,
		TargetAudience: input.TargetAudience,
	}
	planner, err := s.researchPlanner(input.Planner)
	if err != nil {
		return ResearchResult{}, err
	}
	plan, err := planner.PlanResearch(ctx, ResearchPlanningInput{
		Topic:       strings.TrimSpace(input.Topic),
		ContentType: contentType,
		TemplateID:  templateID,
		Sources:     sources,
	})
	if err != nil {
		return ResearchResult{}, err
	}
	initialBrief := draftBriefFromResearch(createInput, plan)
	created, err := s.createRunWithBrief(ctx, createInput, contentType, template, initialBrief, fmt.Sprintf("- v1 created from research topic %q.\n", strings.TrimSpace(input.Topic)))
	if err != nil {
		return ResearchResult{}, err
	}
	fail := func(err error) (ResearchResult, error) {
		s.cleanupCreatedRun(ctx, created.Run.ID)
		return ResearchResult{}, err
	}

	researchPath := s.Files.ResearchPath(created.Run.ID)
	if err := runfiles.WriteJSON(researchPath, plan); err != nil {
		return fail(err)
	}
	checksum, err := runfiles.Checksum(researchPath)
	if err != nil {
		return fail(err)
	}
	artifact := storage.Artifact{
		ID:             s.newID("artifact", "research-json"),
		RunID:          created.Run.ID,
		BriefVersionID: created.Brief.ID,
		Kind:           "research_json",
		Path:           filepath.Clean(researchPath),
		Checksum:       checksum,
	}
	if err := s.Repo.CreateArtifact(ctx, artifact); err != nil {
		return fail(err)
	}
	storedArtifact, err := s.Repo.GetArtifact(ctx, artifact.ID)
	if err != nil {
		return fail(err)
	}
	if err := runfiles.AppendText(created.HistoryPath, fmt.Sprintf("- research plan stored from %d source URLs.\n", len(sources))); err != nil {
		return fail(err)
	}
	return ResearchResult{CreateResult: created, ResearchPath: researchPath, ResearchArtifact: storedArtifact}, nil
}

func (s Service) ListRuns(ctx context.Context) ([]storage.ContentRun, error) {
	return s.Repo.ListContentRuns(ctx)
}

func (s Service) ListTemplates(ctx context.Context) ([]templates.Manifest, error) {
	catalog, err := s.templateCatalog()
	if err != nil {
		return nil, err
	}
	return catalog.All(), nil
}

func (s Service) GetTemplate(ctx context.Context, id string) (templates.Manifest, error) {
	catalog, err := s.templateCatalog()
	if err != nil {
		return templates.Manifest{}, err
	}
	return catalog.Get(id)
}

func (s Service) ShowRun(ctx context.Context, runID string) (RunDetails, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return RunDetails{}, invalidInput("run id is required")
	}
	run, err := s.Repo.GetContentRun(ctx, runID)
	if err != nil {
		return RunDetails{}, err
	}
	briefs, err := s.Repo.ListBriefVersions(ctx, runID)
	if err != nil {
		return RunDetails{}, err
	}
	revisions, err := s.Repo.ListRevisionEventsByRun(ctx, runID)
	if err != nil {
		return RunDetails{}, err
	}
	artifacts, err := s.Repo.ListArtifactsByRun(ctx, runID)
	if err != nil {
		return RunDetails{}, err
	}
	schedules, err := s.Repo.ListScheduledPostsByRun(ctx, runID)
	if err != nil {
		return RunDetails{}, err
	}
	return RunDetails{Run: run, Briefs: briefs, Revisions: revisions, Artifacts: artifacts, Schedules: schedules}, nil
}

func (s Service) ListBriefs(ctx context.Context, runID string) ([]storage.BriefVersion, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, invalidInput("run id is required")
	}
	if _, err := s.Repo.GetContentRun(ctx, runID); err != nil {
		return nil, err
	}
	return s.Repo.ListBriefVersions(ctx, runID)
}

func (s Service) ListArtifacts(ctx context.Context, runID string) ([]storage.Artifact, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, invalidInput("run id is required")
	}
	if _, err := s.Repo.GetContentRun(ctx, runID); err != nil {
		return nil, err
	}
	return s.Repo.ListArtifactsByRun(ctx, runID)
}

func (s Service) GetArtifact(ctx context.Context, runID, artifactID string) (storage.Artifact, error) {
	runID = strings.TrimSpace(runID)
	artifactID = strings.TrimSpace(artifactID)
	if runID == "" {
		return storage.Artifact{}, invalidInput("run id is required")
	}
	if artifactID == "" {
		return storage.Artifact{}, invalidInput("artifact id is required")
	}
	return s.Repo.GetArtifactByRun(ctx, runID, artifactID)
}

func (s Service) DeleteRun(ctx context.Context, runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return invalidInput("run id is required")
	}
	if _, err := s.Repo.GetContentRun(ctx, runID); err != nil {
		return err
	}
	if err := s.Repo.DeleteContentRun(ctx, runID); err != nil {
		return err
	}
	return s.Files.RemoveRun(runID)
}

func (s Service) ReviseRun(ctx context.Context, runID, instruction string) (ReviseResult, error) {
	runID = strings.TrimSpace(runID)
	instruction = strings.TrimSpace(instruction)
	if runID == "" {
		return ReviseResult{}, invalidInput("run id is required")
	}
	if instruction == "" {
		return ReviseResult{}, invalidInput("revision instruction is required")
	}
	run, err := s.Repo.GetContentRun(ctx, runID)
	if err != nil {
		return ReviseResult{}, err
	}
	previous, err := s.Repo.GetLatestBriefVersion(ctx, runID)
	if err != nil {
		return ReviseResult{}, err
	}

	var b brief.Brief
	if err := json.Unmarshal([]byte(previous.BodyJSON), &b); err != nil {
		return ReviseResult{}, err
	}
	b.Hook = fmt.Sprintf("%s Revision: %s", b.Hook, instruction)
	b.CaptionDraft = strings.TrimSpace(b.CaptionDraft + "\n\nRevision note: " + instruction)

	nextVersion := previous.Version + 1
	body, err := marshalBrief(b)
	if err != nil {
		return ReviseResult{}, err
	}
	next := storage.BriefVersion{ID: briefID(runID, nextVersion), RunID: runID, Version: nextVersion, BodyJSON: body}
	if err := s.Repo.CreateBriefVersion(ctx, next); err != nil {
		return ReviseResult{}, err
	}
	storedNext, err := s.Repo.GetBriefVersion(ctx, next.ID)
	if err != nil {
		return ReviseResult{}, err
	}
	event := storage.RevisionEvent{
		ID:                 revisionID(runID, nextVersion),
		RunID:              runID,
		FromBriefVersionID: previous.ID,
		ToBriefVersionID:   next.ID,
		Instruction:        instruction,
	}
	if err := s.Repo.CreateRevisionEvent(ctx, event); err != nil {
		return ReviseResult{}, err
	}

	briefPath := s.Files.BriefPath(runID, nextVersion)
	if err := runfiles.WriteJSON(briefPath, b); err != nil {
		return ReviseResult{}, err
	}
	historyEntry := fmt.Sprintf("- v%d revised from v%d: %s\n", nextVersion, previous.Version, instruction)
	if err := runfiles.AppendText(s.Files.HistoryPath(runID), historyEntry); err != nil {
		return ReviseResult{}, err
	}
	return ReviseResult{Run: run, Previous: previous, Brief: storedNext, BriefPath: briefPath}, nil
}

func (s Service) RenderRun(ctx context.Context, runID string) (RenderResult, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return RenderResult{}, invalidInput("run id is required")
	}
	run, err := s.Repo.GetContentRun(ctx, runID)
	if err != nil {
		return RenderResult{}, err
	}
	latest, err := s.Repo.GetLatestBriefVersion(ctx, runID)
	if err != nil {
		return RenderResult{}, err
	}
	var b brief.Brief
	if err := json.Unmarshal([]byte(latest.BodyJSON), &b); err != nil {
		return RenderResult{}, err
	}

	var content any
	var files []render.OutputFile
	switch run.ContentType {
	case ContentTypeCarousel:
		carousel := render.BuildCarouselContent(b, run.TemplateID)
		content = carousel
		if err := runfiles.WriteJSON(s.Files.ContentPath(runID), carousel); err != nil {
			return RenderResult{}, err
		}
		files, err = s.Carousel.Render(ctx, carousel, s.Files.CarouselOutputDir(runID))
	case ContentTypeShortVideo:
		video := render.BuildVideoContent(b, run.TemplateID)
		content = video
		if err := runfiles.WriteJSON(s.Files.ContentPath(runID), video); err != nil {
			return RenderResult{}, err
		}
		files, err = s.Video.Render(ctx, video, s.Files.VideoOutputDir(runID))
	default:
		err = fmt.Errorf("unsupported content type %q", run.ContentType)
	}
	if err != nil {
		return RenderResult{}, err
	}

	contentPath := s.Files.ContentPath(runID)
	files = append([]render.OutputFile{{Kind: "content_json", Path: contentPath}}, files...)
	// Once renderer files exist, finish metadata writes even if the caller
	// disconnects so disk artifacts and SQLite rows do not drift apart.
	metadataCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	completedAt := s.now()
	job := storage.RenderJob{
		ID:          s.newID("render", runID),
		RunID:       runID,
		Status:      "completed",
		Renderer:    rendererName(run.ContentType),
		CompletedAt: &completedAt,
	}
	if err := s.Repo.CreateRenderJob(metadataCtx, job); err != nil {
		return RenderResult{}, err
	}

	artifactIDs := make([]string, 0, len(files))
	for i, file := range files {
		checksum, err := runfiles.Checksum(file.Path)
		if err != nil {
			return RenderResult{}, err
		}
		artifactID := s.newID("artifact", fmt.Sprintf("%s-%d", file.Kind, i+1))
		artifact := storage.Artifact{
			ID:             artifactID,
			RunID:          runID,
			BriefVersionID: latest.ID,
			Kind:           file.Kind,
			Path:           filepath.Clean(file.Path),
			Checksum:       checksum,
		}
		if err := s.Repo.CreateArtifact(metadataCtx, artifact); err != nil {
			return RenderResult{}, err
		}
		artifactIDs = append(artifactIDs, artifactID)
	}
	artifacts, err := s.Repo.ListArtifactsByIDs(metadataCtx, artifactIDs)
	if err != nil {
		return RenderResult{}, err
	}
	if err := runfiles.AppendText(s.Files.HistoryPath(runID), fmt.Sprintf("- rendered %s artifacts from brief v%d.\n", run.ContentType, latest.Version)); err != nil {
		return RenderResult{}, err
	}

	return RenderResult{Run: run, Brief: latest, Content: content, Artifacts: artifacts}, nil
}

func (s Service) SchedulePost(ctx context.Context, input ScheduleInput) (ScheduleResult, error) {
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		return ScheduleResult{}, invalidInput("run id is required")
	}
	platform := normalizePublishPlatform(input.Platform)
	if platform == "" {
		return ScheduleResult{}, invalidInput("platform is required")
	}
	if !supportedPublishPlatform(platform) {
		return ScheduleResult{}, invalidInput(fmt.Sprintf("unsupported publishing platform %q", input.Platform))
	}
	if input.ScheduledAt.IsZero() {
		return ScheduleResult{}, invalidInput("scheduled_at is required")
	}
	scheduledAt := input.ScheduledAt.UTC()
	if !scheduledAt.After(s.now().UTC()) {
		return ScheduleResult{}, invalidInput("scheduled_at must be in the future")
	}
	run, err := s.Repo.GetContentRun(ctx, runID)
	if err != nil {
		return ScheduleResult{}, err
	}
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID != "" {
		artifact, err := s.Repo.GetArtifact(ctx, artifactID)
		if err != nil {
			return ScheduleResult{}, err
		}
		if artifact.RunID != runID {
			return ScheduleResult{}, invalidInput("artifact does not belong to run")
		}
	}
	validation := publishValidationJSON(run, platform, artifactID)
	post := storage.ScheduledPost{
		ID:             s.newID("schedule", runID+"-"+platform),
		RunID:          runID,
		ArtifactID:     artifactID,
		Platform:       platform,
		ScheduledAt:    scheduledAt,
		Status:         "scheduled",
		ValidationJSON: validation,
	}
	if err := s.Repo.CreateScheduledPost(ctx, post); err != nil {
		return ScheduleResult{}, err
	}
	stored, err := s.Repo.GetScheduledPost(ctx, post.ID)
	if err != nil {
		return ScheduleResult{}, err
	}
	if err := runfiles.AppendText(s.Files.HistoryPath(runID), fmt.Sprintf("- scheduled %s post for %s.\n", platform, stored.ScheduledAt.Format(time.RFC3339))); err != nil {
		return ScheduleResult{}, err
	}
	return ScheduleResult{Run: run, Post: stored}, nil
}

func (s Service) ListScheduledPosts(ctx context.Context) ([]storage.ScheduledPost, error) {
	return s.Repo.ListScheduledPosts(ctx)
}

func (s Service) researchPlanner(requested string) (ResearchPlanner, error) {
	switch strings.ToLower(strings.TrimSpace(requested)) {
	case "", "auto":
		return DeterministicResearchPlanner{}, nil
	case "deterministic":
		return DeterministicResearchPlanner{}, nil
	case "openai":
		switch planner := s.Planner.(type) {
		case OpenAIResearchPlanner:
			return planner, nil
		case *OpenAIResearchPlanner:
			return planner, nil
		default:
			return nil, invalidInput("OPENAI_API_KEY and OPENAI_MODEL are required for --planner openai")
		}
	default:
		return nil, invalidInput(fmt.Sprintf("unsupported research planner %q", requested))
	}
}

type DeterministicResearchPlanner struct{}

func (DeterministicResearchPlanner) PlanResearch(_ context.Context, input ResearchPlanningInput) (ResearchPlan, error) {
	return draftResearchPlan(input.Topic, input.ContentType, input.TemplateID, input.Sources), nil
}

type OpenAIResearchPlanner struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func (p OpenAIResearchPlanner) PlanResearch(ctx context.Context, input ResearchPlanningInput) (ResearchPlan, error) {
	apiKey := strings.TrimSpace(p.APIKey)
	model := strings.TrimSpace(p.Model)
	if apiKey == "" {
		return ResearchPlan{}, invalidInput("OPENAI_API_KEY is required for the openai research planner")
	}
	if model == "" {
		return ResearchPlan{}, invalidInput("OPENAI_MODEL is required for the openai research planner")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	store := false
	body, err := json.Marshal(openAIResearchRequest{
		Model:        model,
		Instructions: openAIResearchInstructions,
		Input:        openAIResearchPrompt(input),
		Store:        &store,
		Text: openAITextConfig{Format: openAITextFormat{
			Type:        "json_schema",
			Name:        "hermeneia_research_plan",
			Description: "A concise Hermeneia research plan for a social content brief.",
			Strict:      true,
			Schema:      researchPlanSchema(),
		}},
	})
	if err != nil {
		return ResearchPlan{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return ResearchPlan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ResearchPlan{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ResearchPlan{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ResearchPlan{}, fmt.Errorf("openai research planner request failed: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	text := extractOpenAIOutputText(data)
	if strings.TrimSpace(text) == "" {
		return ResearchPlan{}, errors.New("openai research planner returned no output text")
	}
	var plan ResearchPlan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return ResearchPlan{}, fmt.Errorf("decode openai research plan: %w", err)
	}
	plan.Topic = strings.TrimSpace(input.Topic)
	plan.ContentType = input.ContentType
	plan.TemplateID = input.TemplateID
	plan.Sources = input.Sources
	plan.Planner = "openai"
	if strings.TrimSpace(plan.Summary) == "" || len(plan.Ideas) == 0 {
		return ResearchPlan{}, errors.New("openai research planner returned an incomplete plan")
	}
	return plan, nil
}

func draftBrief(input CreateInput, contentType, templateID string) brief.Brief {
	topic := strings.TrimSpace(input.Topic)
	tone := strings.TrimSpace(input.Tone)
	if tone == "" {
		tone = "clear, practical, editorial"
	}
	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = "instagram"
	}
	audience := strings.TrimSpace(input.TargetAudience)
	if audience == "" {
		audience = "content operators and social media audiences"
	}
	return brief.Brief{
		Topic:           topic,
		Angle:           "A practical, inspectable take on " + topic,
		Hook:            "What people should understand about " + topic,
		TargetAudience:  audience,
		Platform:        platform,
		ContentType:     contentType,
		Tone:            tone,
		KeyPoints:       []string{"Why this topic matters now", "What changed and who it affects", "How to turn the idea into a useful next action"},
		VisualDirection: "Clean editorial layout with strong contrast, readable hierarchy, and deterministic template output",
		CTA:             "Save this and use it as a starting point for the next content run.",
		CaptionDraft:    "A structured first draft about " + topic + ", ready for review and revision.",
		Hashtags:        []string{"#Hermeneia", "#ContentWorkflow", "#AI"},
	}
}

func draftBriefFromResearch(input CreateInput, plan ResearchPlan) brief.Brief {
	b := draftBrief(input, plan.ContentType, plan.TemplateID)
	b.Angle = "A source-backed editorial take on " + plan.Topic
	b.Hook = "The signal behind " + plan.Topic
	b.KeyPoints = make([]string, 0, len(plan.Ideas)+1)
	b.KeyPoints = append(b.KeyPoints, plan.Summary)
	for _, idea := range plan.Ideas {
		b.KeyPoints = append(b.KeyPoints, idea.Title+": "+idea.Reason)
	}
	b.VisualDirection = "Source-backed editorial layout with clear citations, readable hierarchy, and traceable claims"
	b.CaptionDraft = "Research-backed brief about " + plan.Topic + ". Review the source URLs in research.json before rendering."
	return b
}

func draftResearchPlan(topic, contentType, templateID string, sources []ResearchSource) ResearchPlan {
	ideas := make([]ResearchIdea, 0, len(sources))
	for i, source := range sources {
		label := source.Title
		if label == "" {
			label = source.URL
		}
		reason := source.Note
		if reason == "" {
			reason = "Source preserved for editorial review and traceability."
		}
		ideas = append(ideas, ResearchIdea{Title: label, Reason: reason, Rank: i + 1})
	}
	return ResearchPlan{
		Topic:       strings.TrimSpace(topic),
		Sources:     sources,
		Summary:     fmt.Sprintf("Research seed for %q using %d traceable source URLs.", strings.TrimSpace(topic), len(sources)),
		Ideas:       ideas,
		ContentType: contentType,
		TemplateID:  templateID,
		Planner:     "deterministic",
	}
}

const openAIResearchInstructions = "You are Hermeneia's research planning assistant. Return only JSON matching the supplied schema. Preserve editorial caution: summarize only the provided source metadata and do not invent facts beyond the URLs, titles, and notes."

type openAIResearchRequest struct {
	Model        string           `json:"model"`
	Instructions string           `json:"instructions"`
	Input        string           `json:"input"`
	Store        *bool            `json:"store,omitempty"`
	Text         openAITextConfig `json:"text"`
}

type openAITextConfig struct {
	Format openAITextFormat `json:"format"`
}

type openAITextFormat struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Strict      bool           `json:"strict"`
	Schema      map[string]any `json:"schema"`
}

func openAIResearchPrompt(input ResearchPlanningInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Topic: %s\nContent type: %s\nTemplate: %s\n\nSources:\n", input.Topic, input.ContentType, input.TemplateID)
	for i, source := range input.Sources {
		fmt.Fprintf(&b, "%d. URL: %s\n", i+1, source.URL)
		if source.Title != "" {
			fmt.Fprintf(&b, "   Title: %s\n", source.Title)
		}
		if source.Note != "" {
			fmt.Fprintf(&b, "   Note: %s\n", source.Note)
		}
	}
	return b.String()
}

func researchPlanSchema() map[string]any {
	sourceSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"url", "note", "title"},
		"properties": map[string]any{
			"url":   map[string]any{"type": "string"},
			"note":  map[string]any{"type": "string"},
			"title": map[string]any{"type": "string"},
		},
	}
	ideaSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"title", "reason", "rank"},
		"properties": map[string]any{
			"title":  map[string]any{"type": "string"},
			"reason": map[string]any{"type": "string"},
			"rank":   map[string]any{"type": "integer"},
		},
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"topic", "sources", "summary", "ideas", "content_type", "template_id", "planner"},
		"properties": map[string]any{
			"topic":        map[string]any{"type": "string"},
			"sources":      map[string]any{"type": "array", "items": sourceSchema},
			"summary":      map[string]any{"type": "string"},
			"ideas":        map[string]any{"type": "array", "items": ideaSchema},
			"content_type": map[string]any{"type": "string"},
			"template_id":  map[string]any{"type": "string"},
			"planner":      map[string]any{"type": "string"},
		},
	}
}

func extractOpenAIOutputText(data []byte) string {
	var response struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return ""
	}
	if response.OutputText != "" {
		return response.OutputText
	}
	for _, item := range response.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				return content.Text
			}
		}
	}
	return ""
}

func normalizeResearchSources(sources []ResearchSource) []ResearchSource {
	out := make([]ResearchSource, 0, len(sources))
	for _, source := range sources {
		source.URL = strings.TrimSpace(source.URL)
		source.Note = strings.TrimSpace(source.Note)
		source.Title = strings.TrimSpace(source.Title)
		if source.URL == "" {
			continue
		}
		out = append(out, source)
	}
	return out
}

func marshalBrief(b brief.Brief) (string, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeContentType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ContentTypeCarousel:
		return ContentTypeCarousel, nil
	case "video", "short-video", ContentTypeShortVideo:
		return ContentTypeShortVideo, nil
	default:
		return "", invalidInput(fmt.Sprintf("unsupported content type %q", value))
	}
}

func (s Service) resolveTemplate(contentType, templateID string) (templates.Manifest, error) {
	catalog, err := s.templateCatalog()
	if err != nil {
		return templates.Manifest{}, err
	}
	templateID = strings.TrimSpace(templateID)
	var manifest templates.Manifest
	if templateID == "" {
		manifest, err = catalog.Default(contentType)
	} else {
		manifest, err = catalog.Get(templateID)
	}
	if err != nil {
		return templates.Manifest{}, invalidInput(err.Error())
	}
	if manifest.ContentType != contentType {
		return templates.Manifest{}, invalidInput(fmt.Sprintf("template %q is for content type %q, not %q", manifest.ID, manifest.ContentType, contentType))
	}
	return manifest, nil
}

func (s Service) templateCatalog() (templates.Catalog, error) {
	catalog := s.Templates
	if catalog.Len() > 0 {
		return catalog, nil
	}
	catalog, err := templates.LoadConfigured()
	if err != nil {
		return templates.Catalog{}, fmt.Errorf("load templates: %w", err)
	}
	return catalog, nil
}

func rendererName(contentType string) string {
	switch contentType {
	case ContentTypeShortVideo:
		return "video/remotion-contract-ffmpeg-mvp"
	default:
		return "carousel/go-png"
	}
}

func normalizePublishPlatform(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func supportedPublishPlatform(platform string) bool {
	switch platform {
	case "instagram", "facebook", "youtube", "tiktok", "linkedin":
		return true
	default:
		return false
	}
}

func publishValidationJSON(run storage.ContentRun, platform, artifactID string) string {
	payload := map[string]any{
		"platform":                    platform,
		"content_type":                run.ContentType,
		"credential_storage":          "external_only",
		"credentials_stored_in_db":    false,
		"requires_platform_connector": true,
	}
	if artifactID == "" {
		payload["artifact_selected"] = false
		payload["warning"] = "No rendered artifact was selected for the scheduled post."
	} else {
		payload["artifact_selected"] = true
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (s Service) newID(prefix, seed string) string {
	if s.NewID != nil {
		return s.NewID(prefix, seed)
	}
	return fmt.Sprintf("%s-%s-%s-%s", prefix, s.now().UTC().Format("20060102-150405"), slug(seed), randomSuffix())
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	if len(out) > 36 {
		return strings.Trim(out[:36], "-")
	}
	return out
}

func randomSuffix() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func briefID(runID string, version int) string {
	return fmt.Sprintf("%s-brief-v%d", runID, version)
}

func revisionID(runID string, version int) string {
	return fmt.Sprintf("%s-revision-v%d", runID, version)
}
