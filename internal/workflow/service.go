package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wauputr4/hermeneia/internal/brief"
	"github.com/wauputr4/hermeneia/internal/render"
	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
)

const (
	ContentTypeCarousel   = "carousel"
	ContentTypeShortVideo = "short_video"
)

type CarouselRenderer interface {
	Render(context.Context, render.CarouselContent, string) ([]render.OutputFile, error)
}

type VideoRenderer interface {
	Render(context.Context, render.VideoContent, string) ([]render.OutputFile, error)
}

type Service struct {
	Repo     *storage.Repository
	Files    runfiles.Store
	Carousel CarouselRenderer
	Video    VideoRenderer
	Now      func() time.Time
	NewID    func(prefix, seed string) string
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

type RunDetails struct {
	Run       storage.ContentRun
	Briefs    []storage.BriefVersion
	Revisions []storage.RevisionEvent
	Artifacts []storage.Artifact
}

func NewService(repo *storage.Repository, files runfiles.Store) Service {
	return Service{
		Repo:     repo,
		Files:    files,
		Carousel: render.CarouselRenderer{},
		Video:    render.VideoRenderer{},
	}
}

func (s Service) CreateRun(ctx context.Context, input CreateInput) (CreateResult, error) {
	contentType, err := normalizeContentType(input.ContentType)
	if err != nil {
		return CreateResult{}, err
	}
	if strings.TrimSpace(input.Topic) == "" {
		return CreateResult{}, errors.New("topic is required")
	}
	templateID := strings.TrimSpace(input.TemplateID)
	if templateID == "" {
		templateID = defaultTemplate(contentType)
	}

	b := draftBrief(input, contentType, templateID)
	return s.createRunWithBrief(ctx, input, contentType, templateID, b, fmt.Sprintf("- v1 created from topic %q.\n", b.Topic))
}

func (s Service) createRunWithBrief(ctx context.Context, input CreateInput, contentType, templateID string, b brief.Brief, historyEntry string) (CreateResult, error) {
	runID := s.newID("run", input.Topic)
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

	if err := s.Repo.EnsureTemplate(ctx, storage.Template{ID: templateID, Name: templateName(templateID), ContentType: contentType}); err != nil {
		return fail(err)
	}
	run := storage.ContentRun{ID: runID, Topic: b.Topic, ContentType: contentType, TemplateID: templateID}
	if err := s.Repo.CreateContentRun(ctx, run); err != nil {
		return fail(err)
	}
	runCreated = true
	version := storage.BriefVersion{ID: briefID(runID, 1), RunID: runID, Version: 1, BodyJSON: body}
	if err := s.Repo.CreateBriefVersion(ctx, version); err != nil {
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
	return CreateResult{Run: run, Brief: version, BriefPath: briefPath, HistoryPath: historyPath}, nil
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
		return ResearchResult{}, errors.New("topic is required")
	}
	sources := normalizeResearchSources(input.Sources)
	if len(sources) == 0 {
		return ResearchResult{}, errors.New("at least one source URL is required")
	}
	templateID := strings.TrimSpace(input.TemplateID)
	if templateID == "" {
		templateID = defaultTemplate(contentType)
	}

	createInput := CreateInput{
		Topic:          input.Topic,
		ContentType:    contentType,
		TemplateID:     templateID,
		Tone:           input.Tone,
		Platform:       input.Platform,
		TargetAudience: input.TargetAudience,
	}
	plan := draftResearchPlan(input.Topic, contentType, templateID, sources)
	initialBrief := draftBriefFromResearch(createInput, plan)
	created, err := s.createRunWithBrief(ctx, createInput, contentType, templateID, initialBrief, fmt.Sprintf("- v1 created from research topic %q.\n", strings.TrimSpace(input.Topic)))
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
	if err := runfiles.AppendText(created.HistoryPath, fmt.Sprintf("- research plan stored from %d source URLs.\n", len(sources))); err != nil {
		return fail(err)
	}
	return ResearchResult{CreateResult: created, ResearchPath: researchPath, ResearchArtifact: artifact}, nil
}

func (s Service) ListRuns(ctx context.Context) ([]storage.ContentRun, error) {
	return s.Repo.ListContentRuns(ctx)
}

func (s Service) ShowRun(ctx context.Context, runID string) (RunDetails, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return RunDetails{}, errors.New("run id is required")
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
	return RunDetails{Run: run, Briefs: briefs, Revisions: revisions, Artifacts: artifacts}, nil
}

func (s Service) DeleteRun(ctx context.Context, runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return errors.New("run id is required")
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
		return ReviseResult{}, errors.New("run id is required")
	}
	if instruction == "" {
		return ReviseResult{}, errors.New("revision instruction is required")
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
	return ReviseResult{Run: run, Previous: previous, Brief: next, BriefPath: briefPath}, nil
}

func (s Service) RenderRun(ctx context.Context, runID string) (RenderResult, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return RenderResult{}, errors.New("run id is required")
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
	completedAt := s.now()
	job := storage.RenderJob{
		ID:          s.newID("render", runID),
		RunID:       runID,
		Status:      "completed",
		Renderer:    rendererName(run.ContentType),
		CompletedAt: &completedAt,
	}
	if err := s.Repo.CreateRenderJob(ctx, job); err != nil {
		return RenderResult{}, err
	}

	artifacts := make([]storage.Artifact, 0, len(files))
	for i, file := range files {
		checksum, err := runfiles.Checksum(file.Path)
		if err != nil {
			return RenderResult{}, err
		}
		artifact := storage.Artifact{
			ID:             s.newID("artifact", fmt.Sprintf("%s-%d", file.Kind, i+1)),
			RunID:          runID,
			BriefVersionID: latest.ID,
			Kind:           file.Kind,
			Path:           filepath.Clean(file.Path),
			Checksum:       checksum,
		}
		if err := s.Repo.CreateArtifact(ctx, artifact); err != nil {
			return RenderResult{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	if err := runfiles.AppendText(s.Files.HistoryPath(runID), fmt.Sprintf("- rendered %s artifacts from brief v%d.\n", run.ContentType, latest.Version)); err != nil {
		return RenderResult{}, err
	}

	return RenderResult{Run: run, Brief: latest, Content: content, Artifacts: artifacts}, nil
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
	}
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
		return "", fmt.Errorf("unsupported content type %q", value)
	}
}

func defaultTemplate(contentType string) string {
	switch contentType {
	case ContentTypeShortVideo:
		return render.TemplateVideoAINewsShort
	default:
		return render.TemplateCarouselAINewsClean
	}
}

func templateName(templateID string) string {
	return strings.ReplaceAll(templateID, "/", " ")
}

func rendererName(contentType string) string {
	switch contentType {
	case ContentTypeShortVideo:
		return "video/remotion-contract-ffmpeg-mvp"
	default:
		return "carousel/go-png"
	}
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
