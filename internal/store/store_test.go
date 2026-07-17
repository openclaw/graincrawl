package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func TestStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graincrawl.db")
	st, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC().Round(0)
	title := "Planning"
	note := model.Note{
		ID:         "note-1",
		Title:      &title,
		Type:       "meeting",
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     model.SourcePrivateAPI,
		LastSeenAt: now,
	}
	if err := st.UpsertNote(ctx, note); err != nil {
		t.Fatal(err)
	}
	percentTitle := "100% Ready"
	percentNote := note
	percentNote.ID = "note-2"
	percentNote.Title = &percentTitle
	if err := st.UpsertNote(ctx, percentNote); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GetNote(ctx, "note-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.ID != note.ID || got.Title == nil || *got.Title != title {
		t.Fatalf("unexpected note: %#v", got)
	}
	results, err := st.SearchNotes(ctx, "planning", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != note.ID {
		t.Fatalf("unexpected search results: %#v", results)
	}
	literalPercent, err := st.SearchNotes(ctx, "%", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(literalPercent) != 1 || literalPercent[0].ID != percentNote.ID {
		t.Fatalf("literal percent search results: %#v", literalPercent)
	}
	literalUnderscore, err := st.SearchNotes(ctx, "_", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(literalUnderscore) != 0 {
		t.Fatalf("literal underscore search results: %#v", literalUnderscore)
	}
}

func TestOpenMigratesVersionOneDeletionColumns(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graincrawl-v1.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	oldSchema := `
CREATE TABLE source_objects (
  source TEXT NOT NULL, kind TEXT NOT NULL, source_id TEXT NOT NULL, document_id TEXT,
  payload_json TEXT NOT NULL, payload_hash TEXT NOT NULL, observed_at TEXT NOT NULL,
  PRIMARY KEY(source, kind, source_id)
);
CREATE TABLE notes (
  id TEXT PRIMARY KEY, title TEXT, type TEXT NOT NULL, status TEXT, created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL, deleted_at TEXT, workspace_id TEXT, calendar_event_id TEXT,
  notes_plain TEXT, notes_markdown TEXT, summary_text TEXT, summary_markdown TEXT,
  source TEXT NOT NULL, payload_hash TEXT, last_seen_at TEXT NOT NULL
);
CREATE TABLE transcript_chunks (
  id TEXT PRIMARY KEY, document_id TEXT NOT NULL, start_timestamp TEXT NOT NULL,
  end_timestamp TEXT NOT NULL, source TEXT NOT NULL, is_final INTEGER NOT NULL,
  transcriber_user_id TEXT, text TEXT NOT NULL, payload_hash TEXT
);
CREATE TABLE document_panels (
  id TEXT PRIMARY KEY, document_id TEXT NOT NULL, title TEXT, template_slug TEXT,
  content_plain TEXT, content_markdown TEXT, content_json TEXT, created_at TEXT NOT NULL,
  updated_at TEXT, last_viewed_at TEXT, ydoc_version INTEGER, ydoc_cached_at TEXT,
  source TEXT NOT NULL
);
CREATE TABLE schema_migrations(version INTEGER NOT NULL);
INSERT INTO schema_migrations(version) VALUES(1);
INSERT INTO notes(id, type, created_at, updated_at, deleted_at, source, last_seen_at)
VALUES('deleted-doc', 'meeting', '2026-07-15T10:00:00Z', '2026-07-15T11:00:00Z', '2026-07-16T12:00:00Z', 'private-api', '2026-07-15T11:00:00Z');
INSERT INTO transcript_chunks(id, document_id, start_timestamp, end_timestamp, source, is_final, text)
VALUES('chunk-1', 'deleted-doc', '2026-07-15T10:00:00Z', '2026-07-15T10:00:01Z', 'mic', 1, 'retained');
INSERT INTO document_panels(id, document_id, created_at, source)
VALUES('panel-1', 'deleted-doc', '2026-07-15T10:00:00Z', 'private-api');
INSERT INTO source_objects(source, kind, source_id, document_id, payload_json, payload_hash, observed_at)
VALUES('private-api', 'document', 'deleted-doc', 'deleted-doc', '{}', 'hash', '2026-07-15T11:00:00Z');`
	if _, err := db.ExecContext(ctx, oldSchema); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for table, columns := range map[string][]string{
		"notes":             {"deleted_at", "deletion_source", "deletion_reason"},
		"transcript_chunks": {"deleted_at", "deletion_source", "deletion_reason"},
		"document_panels":   {"deleted_at", "deletion_source", "deletion_reason"},
		"source_objects":    {"deleted_at", "deletion_source", "deletion_reason"},
	} {
		for _, column := range columns {
			var count int
			if err := st.DB().QueryRowContext(ctx, "SELECT count(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("missing migrated column %s.%s", table, column)
			}
		}
	}
	var version int
	if err := st.DB().QueryRowContext(ctx, "SELECT max(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, SchemaVersion)
	}
	for table := range map[string]struct{}{
		"notes": {}, "transcript_chunks": {}, "document_panels": {}, "source_objects": {},
	} {
		where := "id = 'deleted-doc'"
		switch table {
		case "transcript_chunks", "document_panels":
			where = "document_id = 'deleted-doc'"
		case "source_objects":
			where = "source_id = 'deleted-doc'"
		}
		var count int
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s AND deleted_at IS NOT NULL AND deletion_source = 'private-api' AND deletion_reason = ?", table, where)
		if err := st.DB().QueryRowContext(ctx, query, DeletionReasonLegacyField).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("legacy tombstone not reconciled in %s", table)
		}
	}
}

func TestChildUpsertsInheritExistingDocumentTombstone(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	if err := st.TombstoneDocument(ctx, "deleted-doc", Deletion{
		At: now, Source: model.SourcePrivateAPI, Reason: DeletionReasonExplicitFeed,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
		ID: "late-chunk", DocumentID: "deleted-doc", StartTimestamp: now,
		EndTimestamp: now.Add(time.Second), Source: "mic", Text: "retained",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertPanel(ctx, model.Panel{
		ID: "late-panel", DocumentID: "deleted-doc", CreatedAt: now, Source: model.SourcePrivateAPI,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertSourceObject(ctx, SourceObject{
		Source: model.SourcePrivateAPI, Kind: "document", SourceID: "late-source",
		DocumentID: "deleted-doc", PayloadJSON: `{}`, PayloadHash: "hash", ObservedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for table := range map[string]struct{}{
		"transcript_chunks": {}, "document_panels": {}, "source_objects": {},
	} {
		var count int
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE document_id = 'deleted-doc' AND deleted_at IS NOT NULL AND deletion_source = 'private-api' AND deletion_reason = ?", table)
		if err := st.DB().QueryRowContext(ctx, query, DeletionReasonExplicitFeed).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("late child did not inherit tombstone in %s", table)
		}
	}
}
