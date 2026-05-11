package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		"hermeneia schedule",
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

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedule", "run-cli", "--platform", "instagram", "--at", "2026-05-10T02:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "scheduled instagram post") {
		t.Fatalf("unexpected schedule output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedule", "--run=run-cli", "--platform=linkedin", "--at=2026-05-10T03:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "scheduled linkedin post") {
		t.Fatalf("unexpected schedule output:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := cmd.run(ctx, []string{"schedules"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "instagram") || !strings.Contains(stdout.String(), "linkedin") || !strings.Contains(stdout.String(), "scheduled") {
		t.Fatalf("unexpected schedules output:\n%s", stdout.String())
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
