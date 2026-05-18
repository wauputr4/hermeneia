package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/wauputr4/hermeneia/internal/httpapi"
	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/workflow"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hermeneia:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := command{
		stdout: os.Stdout,
	}
	return cmd.run(ctx, args)
}

type command struct {
	stdout   io.Writer
	runsRoot string
	now      func() time.Time
	newID    func(prefix, seed string) string
}

func (c command) run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		c.printUsage()
		return nil
	}

	name := args[0]
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("unknown flag %q; run \"hermeneia help\" for usage", name)
	}

	switch args[0] {
	case "init":
		if len(args) > 1 {
			return fmt.Errorf("init does not accept arguments yet; configure the database with HERMENEIA_DATABASE_PATH")
		}
		path := storage.DatabasePathFromEnv()
		db, err := storage.Open(path)
		if err != nil {
			return fmt.Errorf("open database %q: %w", path, err)
		}
		defer db.Close()
		if err := storage.Migrate(ctx, db); err != nil {
			return fmt.Errorf("migrate database %q: %w", path, err)
		}
		fmt.Fprintf(c.stdout, "initialized Hermeneia database at %s\n", path)
		return nil
	case "create":
		return c.create(ctx, args[1:])
	case "research":
		return c.research(ctx, args[1:])
	case "templates":
		return c.templates(ctx, args[1:])
	case "workflows":
		return c.workflows(ctx, args[1:])
	case "list":
		return c.list(ctx, args[1:])
	case "show":
		return c.show(ctx, args[1:])
	case "revise":
		return c.revise(ctx, args[1:])
	case "render":
		return c.render(ctx, args[1:])
	case "audit":
		return c.audit(ctx, args[1:])
	case "schedule":
		return c.schedule(ctx, args[1:])
	case "cancel-schedule":
		return c.cancelSchedule(ctx, args[1:])
	case "schedules":
		return c.schedules(ctx, args[1:])
	case "serve":
		return c.serve(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q; run \"hermeneia help\" for usage", args[0])
	}
}

func (c command) create(ctx context.Context, args []string) error {
	fs := c.flagSet("create")
	var input workflow.CreateInput
	var workflowID string
	var sources sourceFlags
	var planner string
	fs.StringVar(&input.Topic, "topic", "", "content topic")
	fs.StringVar(&input.ContentType, "type", "carousel", "content type: carousel or short_video")
	fs.StringVar(&input.TemplateID, "template", "", "template id")
	fs.StringVar(&input.Tone, "tone", "", "brief tone")
	fs.StringVar(&input.Platform, "platform", "", "target platform")
	fs.StringVar(&input.TargetAudience, "audience", "", "target audience")
	fs.StringVar(&workflowID, "workflow", "", "workflow preset id")
	fs.Var(&sources, "source", "source URL for workflow presets that require research")
	fs.StringVar(&planner, "planner", "", "research planner for workflow presets")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if input.Topic == "" && fs.NArg() > 0 {
		input.Topic = strings.Join(fs.Args(), " ")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		if strings.TrimSpace(workflowID) != "" {
			result, err := s.CreateRunFromWorkflowPreset(ctx, workflow.WorkflowRunInput{
				WorkflowID:     workflowID,
				Topic:          input.Topic,
				Tone:           input.Tone,
				Platform:       input.Platform,
				TargetAudience: input.TargetAudience,
				Sources:        sources.researchSources(),
				Planner:        planner,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(c.stdout, "created workflow run %s\nbrief %s\n", result.Run.ID, result.BriefPath)
			if result.ResearchPath != "" {
				fmt.Fprintf(c.stdout, "research %s\n", result.ResearchPath)
			}
			if len(result.Artifacts) > 0 {
				w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
				for _, artifact := range result.Artifacts {
					fmt.Fprintf(w, "-\t%s\t%s\n", artifact.Kind, artifact.Path)
				}
				return w.Flush()
			}
			return nil
		}
		result, err := s.CreateRun(ctx, input)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "created run %s\nbrief %s\n", result.Run.ID, result.BriefPath)
		return nil
	})
}

func (c command) research(ctx context.Context, args []string) error {
	input, err := parseResearchArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.CreateRunFromResearch(ctx, input)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "created research run %s\nresearch %s\nbrief %s\n", result.Run.ID, result.ResearchPath, result.BriefPath)
		return nil
	})
}

func (c command) workflows(ctx context.Context, args []string) error {
	fs := c.flagSet("workflows")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("workflows does not accept positional arguments")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		presets, err := s.ListWorkflowPresets(ctx)
		if err != nil {
			return err
		}
		if len(presets) == 0 {
			fmt.Fprintln(c.stdout, "no workflow presets found")
			return nil
		}
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tTEMPLATE\tSTEPS\tDESCRIPTION")
		for _, preset := range presets {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", preset.ID, preset.ContentType, preset.DefaultTemplateID, len(preset.Steps), preset.Description)
		}
		return w.Flush()
	})
}

func (c command) templates(ctx context.Context, args []string) error {
	fs := c.flagSet("templates")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("templates does not accept positional arguments")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		manifests, err := s.ListTemplates(ctx)
		if err != nil {
			return err
		}
		if len(manifests) == 0 {
			fmt.Fprintln(c.stdout, "no templates found")
			return nil
		}
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tRENDERER\tVERSION\tDESCRIPTION")
		for _, manifest := range manifests {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", manifest.ID, manifest.ContentType, manifest.Renderer, manifest.Version, manifest.Description)
		}
		return w.Flush()
	})
}

func (c command) list(ctx context.Context, args []string) error {
	fs := c.flagSet("list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("list does not accept positional arguments")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		runs, err := s.ListRuns(ctx)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			fmt.Fprintln(c.stdout, "no content runs found")
			return nil
		}
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RUN ID\tTYPE\tTEMPLATE\tTOPIC")
		for _, run := range runs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", run.ID, run.ContentType, run.TemplateID, run.Topic)
		}
		return w.Flush()
	})
}

func (c command) show(ctx context.Context, args []string) error {
	runID, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		details, err := s.ShowRun(ctx, runID)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "run:\t%s\n", details.Run.ID)
		fmt.Fprintf(w, "topic:\t%s\n", details.Run.Topic)
		fmt.Fprintf(w, "type:\t%s\n", details.Run.ContentType)
		fmt.Fprintf(w, "template:\t%s\n", details.Run.TemplateID)
		fmt.Fprintf(w, "brief_versions:\t%d\n", len(details.Briefs))
		fmt.Fprintf(w, "revisions:\t%d\n", len(details.Revisions))
		fmt.Fprintf(w, "artifacts:\t%d\n", len(details.Artifacts))
		for _, artifact := range details.Artifacts {
			fmt.Fprintf(w, "-\t%s\t%s\n", artifact.Kind, artifact.Path)
		}
		return w.Flush()
	})
}

func (c command) revise(ctx context.Context, args []string) error {
	runID, instruction, err := parseReviseArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.ReviseRun(ctx, runID, instruction)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "revised run %s to brief v%d\nbrief %s\n", result.Run.ID, result.Brief.Version, result.BriefPath)
		return nil
	})
}

func (c command) render(ctx context.Context, args []string) error {
	runID, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.RenderRun(ctx, runID)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "rendered run %s from brief v%d\n", result.Run.ID, result.Brief.Version)
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		for _, artifact := range result.Artifacts {
			fmt.Fprintf(w, "-\t%s\t%s\n", artifact.Kind, artifact.Path)
		}
		return w.Flush()
	})
}

func (c command) audit(ctx context.Context, args []string) error {
	runID, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.AuditRunArtifacts(ctx, runID)
		if err == nil {
			fmt.Fprintf(c.stdout, "artifact audit passed for %s\n", result.Run.ID)
			return nil
		}
		var auditErr workflow.ArtifactAuditError
		if !errors.As(err, &auditErr) {
			return err
		}
		fmt.Fprintf(c.stdout, "artifact audit failed for %s\n", result.Run.ID)
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KIND\tARTIFACT\tPATH\tMESSAGE")
		for _, issue := range auditErr.Issues {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Kind, issue.ArtifactID, issue.Path, issue.Message)
		}
		if flushErr := w.Flush(); flushErr != nil {
			return flushErr
		}
		return err
	})
}

func (c command) schedule(ctx context.Context, args []string) error {
	input, err := parseScheduleArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.SchedulePost(ctx, input)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "scheduled %s post %s for run %s at %s\n", result.Post.Platform, result.Post.ID, result.Run.ID, result.Post.ScheduledAt.Format(time.RFC3339))
		return nil
	})
}

func (c command) schedules(ctx context.Context, args []string) error {
	fs := c.flagSet("schedules")
	var input workflow.ScheduleListInput
	var jsonOutput bool
	fs.StringVar(&input.Status, "status", "", "filter by schedule status")
	fs.StringVar(&input.Platform, "platform", "", "filter by publishing platform")
	fs.StringVar(&input.From, "from", "", "filter scheduled_at at or after this RFC3339 timestamp")
	fs.StringVar(&input.To, "to", "", "filter scheduled_at at or before this RFC3339 timestamp")
	fs.BoolVar(&jsonOutput, "json", false, "print scheduled posts as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("schedules does not accept positional arguments")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		posts, err := s.ListScheduledPostsFiltered(ctx, input)
		if err != nil {
			return err
		}
		if jsonOutput {
			encoder := json.NewEncoder(c.stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(scheduleJSONRows(posts))
		}
		if len(posts) == 0 {
			fmt.Fprintln(c.stdout, "no scheduled posts found")
			return nil
		}
		w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SCHEDULE ID\tRUN ID\tPLATFORM\tSTATUS\tSCHEDULED AT")
		for _, post := range posts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", post.ID, post.RunID, post.Platform, post.Status, post.ScheduledAt.Format(time.RFC3339))
		}
		return w.Flush()
	})
}

type scheduleJSONRow struct {
	ID          string         `json:"id"`
	RunID       string         `json:"run_id"`
	ArtifactID  string         `json:"artifact_id"`
	Platform    string         `json:"platform"`
	Status      string         `json:"status"`
	ScheduledAt time.Time      `json:"scheduled_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Validation  map[string]any `json:"validation"`
}

func scheduleJSONRows(posts []storage.ScheduledPost) []scheduleJSONRow {
	rows := make([]scheduleJSONRow, 0, len(posts))
	for _, post := range posts {
		rows = append(rows, scheduleJSONRow{
			ID:          post.ID,
			RunID:       post.RunID,
			ArtifactID:  post.ArtifactID,
			Platform:    post.Platform,
			Status:      post.Status,
			ScheduledAt: post.ScheduledAt,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			Validation:  scheduleValidationObject(post.ValidationJSON),
		})
	}
	return rows
}

func scheduleValidationObject(value string) map[string]any {
	if strings.TrimSpace(value) == "" {
		return map[string]any{}
	}
	var validation map[string]any
	if err := json.Unmarshal([]byte(value), &validation); err != nil || validation == nil {
		return map[string]any{}
	}
	return validation
}

func (c command) cancelSchedule(ctx context.Context, args []string) error {
	fs := c.flagSet("cancel-schedule")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("cancel-schedule requires exactly one schedule id")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		result, err := s.UpdateScheduledPostStatus(ctx, workflow.ScheduleStatusInput{
			ScheduleID: fs.Arg(0),
			Status:     "cancelled",
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "cancelled scheduled %s post %s for run %s\n", result.Post.Platform, result.Post.ID, result.Run.ID)
		return nil
	})
}

func (c command) serve(ctx context.Context, args []string) error {
	fs := c.flagSet("serve")
	addr := fs.String("addr", "127.0.0.1:19318", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("serve does not accept positional arguments")
	}
	return c.withService(ctx, func(s *workflow.Service) error {
		server := &http.Server{Addr: *addr, Handler: httpapi.New(s)}
		fmt.Fprintf(c.stdout, "serving Hermeneia API at http://%s\n", *addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
}

func (c command) withService(ctx context.Context, fn func(*workflow.Service) error) error {
	path := storage.DatabasePathFromEnv()
	db, err := storage.Open(path)
	if err != nil {
		return fmt.Errorf("open database %q: %w", path, err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database %q: %w", path, err)
	}
	service := workflow.NewService(storage.NewRepository(db), runfiles.New(c.runsRoot))
	service.Now = c.now
	service.NewID = c.newID
	service.Planner = researchPlannerFromEnv()
	return fn(&service)
}

func (c command) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseRunArgs(args []string) (string, error) {
	var runID string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--run="); ok {
			runID = value
			continue
		}
		if arg == "--run" {
			i++
			if i >= len(args) {
				return "", errors.New("--run requires a value")
			}
			runID = args[i]
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", fmt.Errorf("unknown flag %q", arg)
		}
		if runID != "" {
			return "", fmt.Errorf("unexpected argument %q", arg)
		}
		runID = arg
	}
	return runID, nil
}

func parseReviseArgs(args []string) (string, string, error) {
	var runID string
	var instruction string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--run="); ok {
			runID = value
			continue
		}
		if arg == "--run" {
			i++
			if i >= len(args) {
				return "", "", errors.New("--run requires a value")
			}
			runID = args[i]
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--instruction="); ok {
			instruction = value
			continue
		}
		if arg == "--instruction" {
			i++
			if i >= len(args) {
				return "", "", errors.New("--instruction requires a value")
			}
			instruction = args[i]
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", "", fmt.Errorf("unknown flag %q", arg)
		}
		if runID != "" {
			return "", "", fmt.Errorf("unexpected argument %q", arg)
		}
		runID = arg
	}
	return runID, instruction, nil
}

func parseResearchArgs(args []string) (workflow.ResearchInput, error) {
	var input workflow.ResearchInput
	var topicParts []string
	topicSetExplicitly := false
	input.ContentType = "carousel"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--topic="); ok {
			input.Topic = value
			topicSetExplicitly = true
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--type="); ok {
			input.ContentType = value
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--template="); ok {
			input.TemplateID = value
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--tone="); ok {
			input.Tone = value
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--platform="); ok {
			input.Platform = value
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--audience="); ok {
			input.TargetAudience = value
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--source="); ok {
			input.Sources = append(input.Sources, workflow.ResearchSource{URL: value})
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--planner="); ok {
			input.Planner = value
			continue
		}
		switch arg {
		case "--topic":
			i++
			if i >= len(args) {
				return input, errors.New("--topic requires a value")
			}
			input.Topic = args[i]
			topicSetExplicitly = true
		case "--type":
			i++
			if i >= len(args) {
				return input, errors.New("--type requires a value")
			}
			input.ContentType = args[i]
		case "--template":
			i++
			if i >= len(args) {
				return input, errors.New("--template requires a value")
			}
			input.TemplateID = args[i]
		case "--tone":
			i++
			if i >= len(args) {
				return input, errors.New("--tone requires a value")
			}
			input.Tone = args[i]
		case "--platform":
			i++
			if i >= len(args) {
				return input, errors.New("--platform requires a value")
			}
			input.Platform = args[i]
		case "--audience":
			i++
			if i >= len(args) {
				return input, errors.New("--audience requires a value")
			}
			input.TargetAudience = args[i]
		case "--source":
			i++
			if i >= len(args) {
				return input, errors.New("--source requires a value")
			}
			input.Sources = append(input.Sources, workflow.ResearchSource{URL: args[i]})
		case "--planner":
			i++
			if i >= len(args) {
				return input, errors.New("--planner requires a value")
			}
			input.Planner = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return input, fmt.Errorf("unknown flag %q", arg)
			}
			topicParts = append(topicParts, arg)
		}
	}
	if topicSetExplicitly && len(topicParts) > 0 {
		return input, fmt.Errorf("unexpected positional argument %q when --topic is set", strings.Join(topicParts, " "))
	}
	if input.Topic == "" && len(topicParts) > 0 {
		input.Topic = strings.Join(topicParts, " ")
	}
	return input, nil
}

type sourceFlags []string

func (s *sourceFlags) String() string {
	return strings.Join(*s, ",")
}

func (s *sourceFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s sourceFlags) researchSources() []workflow.ResearchSource {
	out := make([]workflow.ResearchSource, 0, len(s))
	for _, value := range s {
		out = append(out, workflow.ResearchSource{URL: value})
	}
	return out
}

func parseScheduleArgs(args []string) (workflow.ScheduleInput, error) {
	var input workflow.ScheduleInput
	positionalRunID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positionalRunID = args[0]
		args = args[1:]
	}
	fs := flag.NewFlagSet("schedule", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&input.RunID, "run", "", "run id")
	fs.StringVar(&input.Platform, "platform", "", "publishing platform")
	fs.StringVar(&input.ArtifactID, "artifact", "", "artifact id")
	var at string
	fs.StringVar(&at, "at", "", "scheduled time in RFC3339")
	if err := fs.Parse(args); err != nil {
		return input, err
	}
	if at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return input, fmt.Errorf("--at must be RFC3339: %w", err)
		}
		input.ScheduledAt = parsed
	}
	if input.RunID == "" {
		input.RunID = positionalRunID
	}
	if input.RunID == "" && fs.NArg() > 0 {
		input.RunID = fs.Arg(0)
	} else if positionalRunID != "" && fs.NArg() > 0 {
		return input, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if fs.NArg() > 1 {
		return input, fmt.Errorf("unexpected argument %q", fs.Arg(1))
	}
	return input, nil
}

func researchPlannerFromEnv() workflow.ResearchPlanner {
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" || strings.TrimSpace(os.Getenv("OPENAI_MODEL")) == "" {
		return workflow.DeterministicResearchPlanner{}
	}
	return workflow.OpenAIResearchPlanner{
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		BaseURL:    os.Getenv("OPENAI_BASE_URL"),
		Model:      os.Getenv("OPENAI_MODEL"),
		HTTPClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c command) printUsage() {
	fmt.Fprintln(c.stdout, `Hermeneia content workflow CLI

Usage:
  hermeneia init              initialize the SQLite database
  hermeneia create            create a content run
  hermeneia research          create a run from traceable research sources
  hermeneia templates         list available templates
  hermeneia workflows         list available workflow presets
  hermeneia list              list content runs
  hermeneia show              show a content run
  hermeneia revise            create a new brief revision
  hermeneia render            render/export run artifacts
  hermeneia audit             audit artifact file integrity for a run
  hermeneia schedule          create a scheduled publishing record
  hermeneia cancel-schedule   mark a scheduled publishing record cancelled
  hermeneia schedules         list scheduled publishing records
  hermeneia serve             run the local HTTP API

Help:
  hermeneia help
  hermeneia --help

Configuration:
  HERMENEIA_DATABASE_PATH  SQLite path (default: data/hermeneia.db)

Examples:
  hermeneia create --topic "AI agents in marketing" --type carousel
  hermeneia create --workflow simple-carousel --topic "AI agents in marketing"
  hermeneia templates
  hermeneia workflows
  hermeneia research --topic "AI agents" --source "https://example.com/news"
  hermeneia revise <run-id> --instruction "Make the hook sharper"
  hermeneia render <run-id>
  hermeneia audit <run-id>
  hermeneia schedule <run-id> --platform instagram --at 2026-05-10T02:00:00Z
  hermeneia schedules --status scheduled --from 2026-05-10T00:00:00Z --to 2026-05-11T00:00:00Z
  hermeneia cancel-schedule <schedule-id>
  hermeneia serve --addr 127.0.0.1:19318`)
}
