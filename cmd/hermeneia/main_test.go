package main

import (
	"bytes"
	"context"
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
		"hermeneia list",
		"hermeneia show",
		"hermeneia revise",
		"hermeneia render",
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

func TestPlannedCommandsReturnClearErrors(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}}

	for _, name := range []string{"create", "list", "show", "revise", "render"} {
		err := cmd.run(context.Background(), []string{name})
		if err == nil {
			t.Fatalf("expected error for %s", name)
		}
		if got := err.Error(); !strings.Contains(got, "not implemented yet") {
			t.Fatalf("unexpected %s error: %q", name, got)
		}
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
