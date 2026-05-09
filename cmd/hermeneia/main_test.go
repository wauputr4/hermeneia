package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		"hermeneia list",
		"hermeneia show",
		"hermeneia revise",
		"hermeneia render",
		"hermeneia serve",
		"HERMENEIA_DATABASE_PATH",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output does not include %q:\n%s", want, output)
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
	if err := cmd.run(ctx, []string{"show", "run-cli"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"run:", "run-cli", "brief_versions:", "2", "artifacts:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("show output missing %q:\n%s", want, stdout.String())
		}
	}
}
