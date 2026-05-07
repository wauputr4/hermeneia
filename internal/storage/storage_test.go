package storage

import (
    "context"
    "database/sql"
    "os"
    "path/filepath"
    "testing"
)

func TestMigrateAndRepositoryCreateRead(t *testing.T) {
    ctx := context.Background()
    db, err := Open(":memory:")
    if err != nil { t.Fatal(err) }
    defer db.Close()
    if err := Migrate(ctx, db); err != nil { t.Fatal(err) }
    repo := NewRepository(db)
    if err := repo.CreateTemplate(ctx, Template{ID:"tpl-1", Name:"Clean Carousel", ContentType:"carousel"}); err != nil { t.Fatal(err) }
    if err := repo.CreateContentRun(ctx, ContentRun{ID:"run-1", Topic:"AI agents", ContentType:"carousel", TemplateID:"tpl-1"}); err != nil { t.Fatal(err) }
    if err := repo.CreateBriefVersion(ctx, BriefVersion{ID:"brief-1", RunID:"run-1", Version:1, BodyJSON:`{"topic":"AI agents"}`}); err != nil { t.Fatal(err) }
    if err := repo.CreateArtifact(ctx, Artifact{ID:"artifact-1", RunID:"run-1", BriefVersionID:"brief-1", Kind:"carousel_png", Path:"runs/run-1/output/slide-1.png"}); err != nil { t.Fatal(err) }
    run, err := repo.GetContentRun(ctx, "run-1"); if err != nil { t.Fatal(err) }
    if run.Topic != "AI agents" || run.TemplateID != "tpl-1" { t.Fatalf("unexpected run: %#v", run) }
    brief, err := repo.GetBriefVersion(ctx, "brief-1"); if err != nil { t.Fatal(err) }
    if brief.Version != 1 || brief.BodyJSON == "" { t.Fatalf("unexpected brief: %#v", brief) }
    artifact, err := repo.GetArtifact(ctx, "artifact-1"); if err != nil { t.Fatal(err) }
    if artifact.Path != "runs/run-1/output/slide-1.png" { t.Fatalf("unexpected artifact: %#v", artifact) }
    if _, err := repo.GetContentRun(ctx, "missing"); err != sql.ErrNoRows { t.Fatalf("expected sql.ErrNoRows, got %v", err) }
}

func TestDatabasePathFromEnv(t *testing.T) {
    t.Setenv("HERMENEIA_DATABASE_PATH", "/tmp/hermeneia-test.db")
    if got := DatabasePathFromEnv(); got != "/tmp/hermeneia-test.db" { t.Fatalf("unexpected path %q", got) }
}

func TestOpenCreatesDatabaseDirectory(t *testing.T) {
    path := filepath.Join(t.TempDir(), "nested", "data", "hermeneia.db")
    db, err := Open(path)
    if err != nil { t.Fatal(err) }
    defer db.Close()
    if _, err := os.Stat(filepath.Dir(path)); err != nil { t.Fatalf("expected database directory to exist: %v", err) }
}

func TestMigrateRecordsSchemaVersion(t *testing.T) {
    ctx := context.Background()
    db, err := Open(":memory:")
    if err != nil { t.Fatal(err) }
    defer db.Close()
    if err := Migrate(ctx, db); err != nil { t.Fatal(err) }
    if err := Migrate(ctx, db); err != nil { t.Fatalf("second migration should be idempotent: %v", err) }
    var version int
    if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil { t.Fatal(err) }
    if version != schemaVersion { t.Fatalf("unexpected schema version %d", version) }
}
