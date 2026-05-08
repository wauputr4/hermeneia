package workflow

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
)

func TestServiceCreateReviseAndRenderCarouselRun(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	service.Now = func() time.Time { return time.Date(2026, 5, 9, 7, 0, 0, 0, time.UTC) }
	ids := 0
	service.NewID = func(prefix, seed string) string {
		if prefix == "run" {
			return "run-ai-agents"
		}
		ids++
		return prefix + "-" + seed + "-" + string(rune('a'+ids))
	}

	created, err := service.CreateRun(ctx, CreateInput{Topic: "AI agents in marketing", ContentType: "carousel"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Run.ID != "run-ai-agents" {
		t.Fatalf("unexpected run id %q", created.Run.ID)
	}
	if _, err := os.Stat(created.BriefPath); err != nil {
		t.Fatal(err)
	}

	revised, err := service.ReviseRun(ctx, created.Run.ID, "Make the hook sharper")
	if err != nil {
		t.Fatal(err)
	}
	if revised.Brief.Version != 2 {
		t.Fatalf("expected v2, got v%d", revised.Brief.Version)
	}
	if _, err := os.Stat(revised.BriefPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(service.Files.BriefPath(created.Run.ID, 1)); err != nil {
		t.Fatalf("previous brief should be preserved: %v", err)
	}

	rendered, err := service.RenderRun(ctx, created.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered.Artifacts) == 0 {
		t.Fatal("expected render artifacts")
	}
	for _, artifact := range rendered.Artifacts {
		if artifact.Checksum == "" {
			t.Fatalf("artifact missing checksum: %#v", artifact)
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatal(err)
		}
	}

	details, err := service.ShowRun(ctx, created.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Briefs) != 2 || len(details.Revisions) != 1 || len(details.Artifacts) == 0 {
		t.Fatalf("unexpected run details: %#v", details)
	}
	history, err := os.ReadFile(filepath.Join(service.Files.RunDir(created.Run.ID), "history.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(history), "Make the hook sharper") {
		t.Fatalf("history missing revision instruction:\n%s", history)
	}
}

func TestServiceShowRunRequiresExistingRun(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	service := NewService(storage.NewRepository(db), runfiles.New(t.TempDir()))
	if _, err := service.ShowRun(ctx, "missing"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
