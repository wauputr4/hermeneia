package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const DefaultDatabasePath = "data/hermeneia.db"
const schemaVersion = 1

func DatabasePathFromEnv() string {
	if path := os.Getenv("HERMENEIA_DATABASE_PATH"); path != "" {
		return path
	}
	return DefaultDatabasePath
}

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if err := ensureDatabaseDir(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func ensureDatabaseDir(path string) error {
	if path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func Migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, schemaMigrationSQL); err != nil {
		return err
	}

	currentVersion, err := currentSchemaVersion(ctx, tx)
	if err != nil {
		return err
	}
	if currentVersion > schemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", currentVersion, schemaVersion)
	}
	if currentVersion < 1 {
		if _, err := tx.ExecContext(ctx, schemaV1SQL); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, schemaVersion); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func currentSchemaVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	var version sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

type ContentRun struct {
	ID, Topic, ContentType, TemplateID string
	CreatedAt                          time.Time
}
type BriefVersion struct {
	ID, RunID string
	Version   int
	BodyJSON  string
	CreatedAt time.Time
}
type Artifact struct {
	ID, RunID, BriefVersionID, Kind, Path, Checksum string
	CreatedAt                                       time.Time
}
type RenderJob struct {
	ID, RunID, Status, Renderer string
	CreatedAt                   time.Time
	CompletedAt                 *time.Time
}
type RevisionEvent struct {
	ID, RunID, FromBriefVersionID, ToBriefVersionID, Instruction string
	CreatedAt                                                    time.Time
}
type Template struct {
	ID, Name, ContentType string
	CreatedAt             time.Time
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateTemplate(ctx context.Context, t Template) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO templates (id, name, content_type) VALUES (?, ?, ?)`, t.ID, t.Name, t.ContentType)
	return err
}
func (r *Repository) EnsureTemplate(ctx context.Context, t Template) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO templates (id, name, content_type) VALUES (?, ?, ?)`, t.ID, t.Name, t.ContentType)
	return err
}
func (r *Repository) GetTemplate(ctx context.Context, id string) (Template, error) {
	var t Template
	err := r.db.QueryRowContext(ctx, `SELECT id, name, content_type, created_at FROM templates WHERE id = ?`, id).Scan(&t.ID, &t.Name, &t.ContentType, &t.CreatedAt)
	return t, err
}
func (r *Repository) CreateContentRun(ctx context.Context, cr ContentRun) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO content_runs (id, topic, content_type, template_id) VALUES (?, ?, ?, ?)`, cr.ID, cr.Topic, cr.ContentType, nullIfEmpty(cr.TemplateID))
	return err
}
func (r *Repository) DeleteContentRun(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM content_runs WHERE id = ?`, id)
	return err
}
func (r *Repository) GetContentRun(ctx context.Context, id string) (ContentRun, error) {
	var cr ContentRun
	var template sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id, topic, content_type, template_id, created_at FROM content_runs WHERE id = ?`, id).Scan(&cr.ID, &cr.Topic, &cr.ContentType, &template, &cr.CreatedAt)
	if template.Valid {
		cr.TemplateID = template.String
	}
	return cr, err
}
func (r *Repository) ListContentRuns(ctx context.Context) ([]ContentRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, topic, content_type, template_id, created_at FROM content_runs ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []ContentRun
	for rows.Next() {
		var cr ContentRun
		var template sql.NullString
		if err := rows.Scan(&cr.ID, &cr.Topic, &cr.ContentType, &template, &cr.CreatedAt); err != nil {
			return nil, err
		}
		if template.Valid {
			cr.TemplateID = template.String
		}
		runs = append(runs, cr)
	}
	return runs, rows.Err()
}
func (r *Repository) CreateBriefVersion(ctx context.Context, bv BriefVersion) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO brief_versions (id, run_id, version, body_json) VALUES (?, ?, ?, ?)`, bv.ID, bv.RunID, bv.Version, bv.BodyJSON)
	return err
}
func (r *Repository) GetBriefVersion(ctx context.Context, id string) (BriefVersion, error) {
	var bv BriefVersion
	err := r.db.QueryRowContext(ctx, `SELECT id, run_id, version, body_json, created_at FROM brief_versions WHERE id = ?`, id).Scan(&bv.ID, &bv.RunID, &bv.Version, &bv.BodyJSON, &bv.CreatedAt)
	return bv, err
}
func (r *Repository) GetLatestBriefVersion(ctx context.Context, runID string) (BriefVersion, error) {
	var bv BriefVersion
	err := r.db.QueryRowContext(ctx, `SELECT id, run_id, version, body_json, created_at FROM brief_versions WHERE run_id = ? ORDER BY version DESC LIMIT 1`, runID).Scan(&bv.ID, &bv.RunID, &bv.Version, &bv.BodyJSON, &bv.CreatedAt)
	return bv, err
}
func (r *Repository) ListBriefVersions(ctx context.Context, runID string) ([]BriefVersion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, run_id, version, body_json, created_at FROM brief_versions WHERE run_id = ? ORDER BY version ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []BriefVersion
	for rows.Next() {
		var bv BriefVersion
		if err := rows.Scan(&bv.ID, &bv.RunID, &bv.Version, &bv.BodyJSON, &bv.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, bv)
	}
	return versions, rows.Err()
}
func (r *Repository) CreateArtifact(ctx context.Context, a Artifact) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO artifacts (id, run_id, brief_version_id, kind, path, checksum) VALUES (?, ?, ?, ?, ?, ?)`, a.ID, a.RunID, nullIfEmpty(a.BriefVersionID), a.Kind, a.Path, nullIfEmpty(a.Checksum))
	return err
}
func (r *Repository) GetArtifact(ctx context.Context, id string) (Artifact, error) {
	var a Artifact
	var bvid, checksum sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id, run_id, brief_version_id, kind, path, checksum, created_at FROM artifacts WHERE id = ?`, id).Scan(&a.ID, &a.RunID, &bvid, &a.Kind, &a.Path, &checksum, &a.CreatedAt)
	if bvid.Valid {
		a.BriefVersionID = bvid.String
	}
	if checksum.Valid {
		a.Checksum = checksum.String
	}
	return a, err
}
func (r *Repository) ListArtifactsByRun(ctx context.Context, runID string) ([]Artifact, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, run_id, brief_version_id, kind, path, checksum, created_at FROM artifacts WHERE run_id = ? ORDER BY created_at ASC, id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []Artifact
	for rows.Next() {
		var a Artifact
		var bvid, checksum sql.NullString
		if err := rows.Scan(&a.ID, &a.RunID, &bvid, &a.Kind, &a.Path, &checksum, &a.CreatedAt); err != nil {
			return nil, err
		}
		if bvid.Valid {
			a.BriefVersionID = bvid.String
		}
		if checksum.Valid {
			a.Checksum = checksum.String
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, rows.Err()
}
func (r *Repository) CreateRenderJob(ctx context.Context, job RenderJob) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO render_jobs (id, run_id, status, renderer, completed_at) VALUES (?, ?, ?, ?, ?)`, job.ID, job.RunID, job.Status, job.Renderer, nullIfNilTime(job.CompletedAt))
	return err
}
func (r *Repository) GetRenderJob(ctx context.Context, id string) (RenderJob, error) {
	var job RenderJob
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id, run_id, status, renderer, created_at, completed_at FROM render_jobs WHERE id = ?`, id).Scan(&job.ID, &job.RunID, &job.Status, &job.Renderer, &job.CreatedAt, &completedAt)
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	return job, err
}
func (r *Repository) CreateRevisionEvent(ctx context.Context, e RevisionEvent) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO revision_events (id, run_id, from_brief_version_id, to_brief_version_id, instruction) VALUES (?, ?, ?, ?, ?)`, e.ID, e.RunID, e.FromBriefVersionID, e.ToBriefVersionID, e.Instruction)
	return err
}
func (r *Repository) GetRevisionEvent(ctx context.Context, id string) (RevisionEvent, error) {
	var e RevisionEvent
	err := r.db.QueryRowContext(ctx, `SELECT id, run_id, from_brief_version_id, to_brief_version_id, instruction, created_at FROM revision_events WHERE id = ?`, id).Scan(&e.ID, &e.RunID, &e.FromBriefVersionID, &e.ToBriefVersionID, &e.Instruction, &e.CreatedAt)
	return e, err
}
func (r *Repository) ListRevisionEventsByRun(ctx context.Context, runID string) ([]RevisionEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, run_id, from_brief_version_id, to_brief_version_id, instruction, created_at FROM revision_events WHERE run_id = ? ORDER BY created_at ASC, id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []RevisionEvent
	for rows.Next() {
		var e RevisionEvent
		if err := rows.Scan(&e.ID, &e.RunID, &e.FromBriefVersionID, &e.ToBriefVersionID, &e.Instruction, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullIfNilTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

const schemaMigrationSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
`

const schemaV1SQL = `
CREATE TABLE IF NOT EXISTS templates (id TEXT PRIMARY KEY, name TEXT NOT NULL, content_type TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS content_runs (id TEXT PRIMARY KEY, topic TEXT NOT NULL, content_type TEXT NOT NULL, template_id TEXT REFERENCES templates(id), created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS brief_versions (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES content_runs(id) ON DELETE CASCADE, version INTEGER NOT NULL, body_json TEXT NOT NULL CHECK (json_valid(body_json)), created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE(run_id, version));
CREATE TABLE IF NOT EXISTS render_jobs (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES content_runs(id) ON DELETE CASCADE, status TEXT NOT NULL, renderer TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, completed_at DATETIME);
CREATE TABLE IF NOT EXISTS artifacts (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES content_runs(id) ON DELETE CASCADE, brief_version_id TEXT REFERENCES brief_versions(id), kind TEXT NOT NULL, path TEXT NOT NULL, checksum TEXT, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS revision_events (id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES content_runs(id) ON DELETE CASCADE, from_brief_version_id TEXT NOT NULL REFERENCES brief_versions(id), to_brief_version_id TEXT NOT NULL REFERENCES brief_versions(id), instruction TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
`
