package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/workflow"
)

func TestHelpOutputIncludesMVPCommandSurface(t *testing.T) {
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout}

	if err := cmd.run(context.Background(), []string{"help"}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{
		"hermeneia init",
		"hermeneia create",
		"hermeneia research",
		"hermeneia templates",
		"hermeneia workflows",
		"hermeneia list",
		"hermeneia show",
		"hermeneia revise",
		"hermeneia render",
		"hermeneia audit",
		"hermeneia schedule",
		"hermeneia cancel-schedule",
		"hermeneia schedules",
		"hermeneia serve",
		"HERMENEIA_DATABASE_PATH",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output does not include %q:\n%s", want, output)
		}
	}
}

func TestCLIWorkflowsListsBuiltInCatalog(t *testing.T) {
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	cmd := command{stdout: &stdout}
	if err := cmd.run(context.Background(), []string{"workflows"}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{"ID", "TYPE", "TEMPLATE", "STEPS", "simple-carousel", "research-carousel", "carousel/ai-news-clean"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workflows output missing %q:\n%s", want, output)
		}
	}
}

func TestCLITemplatesListsBuiltInCatalog(t *testing.T) {
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	cmd := command{stdout: &stdout}
	if err := cmd.run(context.Background(), []string{"templates"}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{"ID", "TYPE", "RENDERER", "VERSION", "carousel/ai-news-clean", "video/ai-news-short"} {
		if !strings.Contains(output, want) {
			t.Fatalf("templates output missing %q:\n%s", want, output)
		}
	}
}

func TestCLITemplatesIncludesCustomTemplatePath(t *testing.T) {
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	customRoot := t.TempDir()
	writeCLITestManifest(t, customRoot, "carousel/custom-local")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)
	t.Setenv("HERMENEIA_TEMPLATE_PATH", customRoot)

	cmd := command{stdout: &stdout}
	if err := cmd.run(context.Background(), []string{"templates"}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{"carousel/ai-news-clean", "video/ai-news-short", "carousel/custom-local"} {
		if !strings.Contains(output, want) {
			t.Fatalf("templates output missing %q:\n%s", want, output)
		}
	}
}

func TestCLICreateWorkflowPresetCreatesRenderedRun(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-workflow-cli"
			}
			ids++
			return prefix + "-workflow-cli-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--workflow", "simple-carousel", "--topic", "AI agents in marketing"}); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, want := range []string{"created workflow run run-workflow-cli", "brief", "carousel_png"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workflow create output missing %q:\n%s", want, output)
		}
	}
	if _, err := os.Stat(filepath.Join(runsRoot, "run-workflow-cli", "output", "carousel", "slide-01.png")); err != nil {
		t.Fatal(err)
	}
}

func TestCLIAuditPassesForRenderedWorkflowRun(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-audit-cli"
			}
			ids++
			return prefix + "-audit-cli-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--workflow", "simple-carousel", "--topic", "AI agents in marketing"}); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := cmd.run(ctx, []string{"audit", "run-audit-cli"}); err != nil {
		t.Fatal(err)
	}
	if output := stdout.String(); !strings.Contains(output, "artifact audit passed for run-audit-cli") {
		t.Fatalf("audit output missing pass message:\n%s", output)
	}
}

func TestCLIAuditJSONPassesForRenderedWorkflowRun(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-audit-json-healthy"
			}
			ids++
			return prefix + "-audit-json-healthy-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--workflow", "simple-carousel", "--topic", "AI agents in marketing"}); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := cmd.run(ctx, []string{"audit", "run-audit-json-healthy", "--json"}); err != nil {
		t.Fatal(err)
	}
	var result artifactAuditJSON
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("audit JSON is invalid: %v\n%s", err, stdout.String())
	}
	if !result.Healthy || result.Run.ID != "run-audit-json-healthy" || len(result.Issues) != 0 {
		t.Fatalf("unexpected audit JSON: %#v", result)
	}
}

func TestCLIAuditJSONPrintsDriftPayloadBeforeReturningError(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-audit-json-drift"
			}
			ids++
			return prefix + "-audit-json-drift-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--workflow", "simple-carousel", "--topic", "AI agents in marketing"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(runsRoot, "run-audit-json-drift", "output", "carousel", "slide-01.png")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	err := cmd.run(ctx, []string{"audit", "--run", "run-audit-json-drift", "--json"})
	if err == nil || !errors.Is(err, workflow.ErrInvalidInput) {
		t.Fatalf("expected audit drift error, got %v", err)
	}
	var result artifactAuditJSON
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("audit drift JSON is invalid: %v\n%s", decodeErr, stdout.String())
	}
	if result.Healthy || result.Run.ID != "run-audit-json-drift" || !artifactAuditJSONContains(result.Issues, "missing_file") {
		t.Fatalf("unexpected audit drift JSON: %#v", result)
	}
}

func TestCLIAuditHumanOutputReportsDriftTable(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-audit-human-drift"
			}
			ids++
			return prefix + "-audit-human-drift-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--workflow", "simple-carousel", "--topic", "AI agents in marketing"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(runsRoot, "run-audit-human-drift", "output", "carousel", "slide-01.png")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	err := cmd.run(ctx, []string{"audit", "run-audit-human-drift"})
	if err == nil || !errors.Is(err, workflow.ErrInvalidInput) {
		t.Fatalf("expected audit drift error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"artifact audit failed for run-audit-human-drift", "KIND", "ARTIFACT", "PATH", "MESSAGE", "missing_file"} {
		if !strings.Contains(output, want) {
			t.Fatalf("audit human output missing %q:\n%s", want, output)
		}
	}
}

func artifactAuditJSONContains(issues []artifactAuditIssue, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

func TestUnknownCommandReturnsClearError(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}}

	err := cmd.run(context.Background(), []string{"publish"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "unknown command") || !strings.Contains(got, "hermeneia help") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestInitCreatesSQLiteDatabase(t *testing.T) {
	var stdout bytes.Buffer
	path := filepath.Join(t.TempDir(), "nested", "hermeneia.db")
	t.Setenv("HERMENEIA_DATABASE_PATH", path)

	cmd := command{stdout: &stdout}
	if err := cmd.run(context.Background(), []string{"init"}); err != nil {
		t.Fatal(err)
	}

	if output := stdout.String(); !strings.Contains(output, path) {
		t.Fatalf("init output does not include database path %q:\n%s", path, output)
	}
}

func TestInitRejectsUnexpectedArguments(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}}

	err := cmd.run(context.Background(), []string{"init", "--force"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "does not accept arguments") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestCLIResearchCreatesTraceableRun(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-research"
			}
			return prefix + "-research"
		},
	}
	err := cmd.run(ctx, []string{
		"research",
		"AI agents in marketing",
		"--source", "https://example.com/agents",
		"--source", "https://example.com/marketing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "created research run run-research") {
		t.Fatalf("unexpected research output:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(runsRoot, "run-research", "research.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCLIResearchRejectsStrayArgsWhenTopicIsSet(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}}

	err := cmd.run(context.Background(), []string{
		"research",
		"--topic", "AI agents",
		"--source", "https://example.com/agents",
		"https://example.com/marketing",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected positional argument") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestCLIResearchOpenAIPlannerRequiresConfig(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")

	cmd := command{stdout: &stdout, runsRoot: runsRoot}
	err := cmd.run(ctx, []string{
		"research",
		"--topic", "AI agents",
		"--source", "https://example.com/agents",
		"--planner", "openai",
	})
	if err == nil {
		t.Fatal("expected missing OpenAI config error")
	}
	if got := err.Error(); !strings.Contains(got, "OPENAI_API_KEY") || !strings.Contains(got, "OPENAI_MODEL") {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestResearchPlannerFromEnvConfiguresReusableHTTPClient(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	t.Setenv("OPENAI_BASE_URL", "https://api.openai.com/v1")

	planner, ok := researchPlannerFromEnv().(workflow.OpenAIResearchPlanner)
	if !ok {
		t.Fatalf("expected OpenAI planner, got %T", researchPlannerFromEnv())
	}
	if planner.HTTPClient == nil {
		t.Fatal("expected reusable HTTP client")
	}
}

func writeCLITestManifest(t *testing.T, root, id string) {
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

func TestCLIContentRunWorkflow(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	runsRoot := filepath.Join(t.TempDir(), "runs")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)

	ids := 0
	cmd := command{
		stdout:   &stdout,
		runsRoot: runsRoot,
		now:      func() time.Time { return time.Date(2026, 5, 9, 7, 0, 0, 0, time.UTC) },
		newID: func(prefix, seed string) string {
			if prefix == "run" {
				return "run-cli"
			}
			ids++
			return prefix + "-cli-" + string(rune('a'+ids))
		},
	}
	if err := cmd.run(ctx, []string{"create", "--topic", "AI agents in marketing", "--type", "carousel"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "created run run-cli") {
		t.Fatalf("unexpected create output:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(runsRoot, "run-cli", "brief.v1.json")); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"revise", "run-cli", "--instruction", "Make the hook sharper"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "brief v2") {
		t.Fatalf("unexpected revise output:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(runsRoot, "run-cli", "brief.v2.json")); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"render", "run-cli"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "carousel_png") {
		t.Fatalf("unexpected render output:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(runsRoot, "run-cli", "output", "carousel", "slide-01.png")); err != nil {
		t.Fatal(err)
	}
	artifactID := firstCLITestArtifactID(t, ctx, dbPath, "run-cli")

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedule", "run-cli", "--platform", "instagram", "--at", "2026-05-10T02:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "scheduled instagram post") {
		t.Fatalf("unexpected schedule output:\n%s", stdout.String())
	}
	instagramScheduleID := strings.Fields(stdout.String())[3]

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedule", "--run=run-cli", "--platform=linkedin", "--artifact=" + artifactID, "--at=2026-05-10T03:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "scheduled linkedin post") {
		t.Fatalf("unexpected schedule output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "ARTIFACT ID") || !strings.Contains(stdout.String(), "instagram") || !strings.Contains(stdout.String(), "linkedin") || !strings.Contains(stdout.String(), "scheduled") {
		t.Fatalf("unexpected schedules output:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), artifactID) || !strings.Contains(stdout.String(), "none") {
		t.Fatalf("schedules output missing artifact id or no-artifact fallback:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--status", "scheduled"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "instagram") || !strings.Contains(stdout.String(), "linkedin") {
		t.Fatalf("expected scheduled rows in status-filtered output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--platform", "instagram"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "instagram") || strings.Contains(stdout.String(), "linkedin") {
		t.Fatalf("unexpected platform-filtered schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--artifact", artifactID}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "linkedin") || strings.Contains(stdout.String(), "instagram") {
		t.Fatalf("unexpected artifact-filtered schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--status", "scheduled", "--platform", "linkedin"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "linkedin") || strings.Contains(stdout.String(), "instagram") {
		t.Fatalf("unexpected combined-filter schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--from", "2026-05-10T03:00:00Z", "--to", "2026-05-10T03:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "linkedin") || strings.Contains(stdout.String(), "instagram") {
		t.Fatalf("unexpected range-filtered schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--json"}); err != nil {
		t.Fatal(err)
	}
	var allScheduleRows []scheduleJSONRow
	if err := json.Unmarshal(stdout.Bytes(), &allScheduleRows); err != nil {
		t.Fatalf("schedules --json did not print valid JSON: %v\n%s", err, stdout.String())
	}
	if len(allScheduleRows) != 2 {
		t.Fatalf("expected two JSON schedule rows, got %#v", allScheduleRows)
	}
	if allScheduleRows[0].ID == "" || allScheduleRows[0].RunID != "run-cli" || allScheduleRows[0].ScheduledAt.IsZero() || allScheduleRows[0].CreatedAt.IsZero() || allScheduleRows[0].UpdatedAt.IsZero() {
		t.Fatalf("JSON schedule row missing stable fields: %#v", allScheduleRows[0])
	}
	if allScheduleRows[0].Validation["credentials_stored_in_db"] != false || allScheduleRows[0].Validation["credential_storage"] != "external_only" {
		t.Fatalf("JSON schedule row missing validation metadata: %#v", allScheduleRows[0])
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--status", "scheduled", "--platform", "instagram", "--json"}); err != nil {
		t.Fatal(err)
	}
	var filteredScheduleRows []scheduleJSONRow
	if err := json.Unmarshal(stdout.Bytes(), &filteredScheduleRows); err != nil {
		t.Fatalf("filtered schedules --json did not print valid JSON: %v\n%s", err, stdout.String())
	}
	if len(filteredScheduleRows) != 1 || filteredScheduleRows[0].Platform != "instagram" || filteredScheduleRows[0].Status != "scheduled" {
		t.Fatalf("unexpected filtered JSON schedule rows: %#v", filteredScheduleRows)
	}
	if filteredScheduleRows[0].Validation["requires_platform_connector"] != true {
		t.Fatalf("filtered JSON schedule row missing validation metadata: %#v", filteredScheduleRows[0])
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--artifact", artifactID, "--status", "scheduled", "--platform", "linkedin", "--json"}); err != nil {
		t.Fatal(err)
	}
	var artifactFilteredScheduleRows []scheduleJSONRow
	if err := json.Unmarshal(stdout.Bytes(), &artifactFilteredScheduleRows); err != nil {
		t.Fatalf("artifact-filtered schedules --json did not print valid JSON: %v\n%s", err, stdout.String())
	}
	if len(artifactFilteredScheduleRows) != 1 || artifactFilteredScheduleRows[0].ArtifactID != artifactID || artifactFilteredScheduleRows[0].Platform != "linkedin" {
		t.Fatalf("unexpected artifact-filtered JSON schedule rows: %#v", artifactFilteredScheduleRows)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--artifact", "missing-artifact"}); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "no scheduled posts found\n" {
		t.Fatalf("unexpected missing-artifact schedules output: %q", got)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--status", "scheduled", "--platform", "linkedin", "--from", "2026-05-10T02:00:00Z", "--to", "2026-05-10T03:00:00Z", "--json"}); err != nil {
		t.Fatal(err)
	}
	var rangeFilteredScheduleRows []scheduleJSONRow
	if err := json.Unmarshal(stdout.Bytes(), &rangeFilteredScheduleRows); err != nil {
		t.Fatalf("range-filtered schedules --json did not print valid JSON: %v\n%s", err, stdout.String())
	}
	if len(rangeFilteredScheduleRows) != 1 || rangeFilteredScheduleRows[0].Platform != "linkedin" || !rangeFilteredScheduleRows[0].ScheduledAt.Equal(time.Date(2026, 5, 10, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected range-filtered JSON schedule rows: %#v", rangeFilteredScheduleRows)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"cancel-schedule", instagramScheduleID}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cancelled scheduled instagram post") {
		t.Fatalf("unexpected cancel-schedule output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cancelled") {
		t.Fatalf("expected cancelled status in schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--status", "cancelled"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "instagram") || !strings.Contains(stdout.String(), "cancelled") || strings.Contains(stdout.String(), "linkedin") {
		t.Fatalf("unexpected cancelled-filter schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	err := cmd.run(ctx, []string{"schedules", "--status", "queued"})
	if err == nil {
		t.Fatal("expected invalid status error")
	}
	if got := err.Error(); !strings.Contains(got, "unsupported scheduled post status") {
		t.Fatalf("unexpected invalid status error: %q", got)
	}
	if stdout.String() != "" {
		t.Fatalf("invalid status printed output before failing: %q", stdout.String())
	}

	stdout.Reset()
	err = cmd.run(ctx, []string{"schedules", "--platform", "mastodon"})
	if err == nil {
		t.Fatal("expected invalid platform error")
	}
	if got := err.Error(); !strings.Contains(got, "unsupported publishing platform") {
		t.Fatalf("unexpected invalid platform error: %q", got)
	}
	if stdout.String() != "" {
		t.Fatalf("invalid platform printed output before failing: %q", stdout.String())
	}

	stdout.Reset()
	err = cmd.run(ctx, []string{"schedules", "--from", "tomorrow"})
	if err == nil {
		t.Fatal("expected invalid from error")
	}
	if got := err.Error(); !strings.Contains(got, "from must be a valid RFC3339 timestamp") {
		t.Fatalf("unexpected invalid from error: %q", got)
	}
	if stdout.String() != "" {
		t.Fatalf("invalid from printed output before failing: %q", stdout.String())
	}

	stdout.Reset()
	err = cmd.run(ctx, []string{"schedules", "--from", "2026-05-10T04:00:00Z", "--to", "2026-05-10T03:00:00Z"})
	if err == nil {
		t.Fatal("expected inverted range error")
	}
	if got := err.Error(); !strings.Contains(got, "from must be before or equal to to") {
		t.Fatalf("unexpected inverted range error: %q", got)
	}
	if stdout.String() != "" {
		t.Fatalf("inverted range printed output before failing: %q", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"show", "run-cli"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"run:", "run-cli", "brief_versions:", "2", "artifacts:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("show output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestScheduleValidationObjectEmptyFallback(t *testing.T) {
	for _, value := range []string{"", "  ", "null", "[]", "{malformed"} {
		validation := scheduleValidationObject(value)
		if len(validation) != 0 {
			t.Fatalf("expected empty validation fallback for %q, got %#v", value, validation)
		}
	}
}

func TestCLISchedulesRunFilter(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "hermeneia.db")
	t.Setenv("HERMENEIA_DATABASE_PATH", dbPath)
	seedCLIScheduleFilterData(t, ctx, dbPath)

	cmd := command{stdout: &stdout}
	if err := cmd.run(ctx, []string{"schedules", "--run", "run-alpha"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "schedule-alpha-instagram") || !strings.Contains(stdout.String(), "schedule-alpha-youtube") || strings.Contains(stdout.String(), "schedule-beta-instagram") {
		t.Fatalf("unexpected run-filtered schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--run", "run-alpha", "--platform", "youtube", "--status", "scheduled", "--from", "2026-05-10T03:00:00Z", "--to", "2026-05-10T03:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "schedule-alpha-youtube") || strings.Contains(stdout.String(), "schedule-alpha-instagram") || strings.Contains(stdout.String(), "schedule-beta-instagram") {
		t.Fatalf("unexpected combined run-filtered schedules output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--run", "run-alpha", "--json"}); err != nil {
		t.Fatal(err)
	}
	var rows []scheduleJSONRow
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("run-filtered schedules --json did not print valid JSON: %v\n%s", err, stdout.String())
	}
	if len(rows) != 2 || rows[0].RunID != "run-alpha" || rows[1].RunID != "run-alpha" {
		t.Fatalf("unexpected run-filtered JSON schedules: %#v", rows)
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules", "--run", "missing-run"}); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "no scheduled posts found\n" {
		t.Fatalf("unexpected missing-run schedules output: %q", got)
	}
}

func firstCLITestArtifactID(t *testing.T, ctx context.Context, dbPath, runID string) string {
	t.Helper()
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	artifacts, err := storage.NewRepository(db).ListArtifactsByRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) == 0 {
		t.Fatal("expected rendered artifacts")
	}
	return artifacts[0].ID
}

func seedCLIScheduleFilterData(t *testing.T, ctx context.Context, dbPath string) {
	t.Helper()
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := storage.NewRepository(db)
	if err := repo.CreateTemplate(ctx, storage.Template{ID: "carousel/ai-news-clean", Name: "AI News Clean Carousel", ContentType: "carousel"}); err != nil {
		t.Fatal(err)
	}
	for _, runID := range []string{"run-alpha", "run-beta"} {
		if err := repo.CreateContentRun(ctx, storage.ContentRun{ID: runID, Topic: runID, ContentType: "carousel", TemplateID: "carousel/ai-news-clean"}); err != nil {
			t.Fatal(err)
		}
	}
	posts := []storage.ScheduledPost{
		{ID: "schedule-alpha-instagram", RunID: "run-alpha", Platform: "instagram", ScheduledAt: time.Date(2026, 5, 10, 2, 0, 0, 0, time.UTC), Status: "scheduled"},
		{ID: "schedule-alpha-youtube", RunID: "run-alpha", Platform: "youtube", ScheduledAt: time.Date(2026, 5, 10, 3, 0, 0, 0, time.UTC), Status: "scheduled"},
		{ID: "schedule-beta-instagram", RunID: "run-beta", Platform: "instagram", ScheduledAt: time.Date(2026, 5, 10, 4, 0, 0, 0, time.UTC), Status: "scheduled"},
	}
	for _, post := range posts {
		if err := repo.CreateScheduledPost(ctx, post); err != nil {
			t.Fatal(err)
		}
	}
}
