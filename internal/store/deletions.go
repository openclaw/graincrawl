package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

const (
	DeletionReasonExplicitFeed = "explicit-delete-feed"
	DeletionReasonSourceField  = "source-deleted-at"
	DeletionReasonLegacyField  = "legacy-deleted-at"
)

type Deletion struct {
	At     time.Time
	Source model.Source
	Reason string
}

// ReconcileTombstones repairs legacy/imported rows whose note had a deletion
// timestamp before child provenance columns existed.
func ReconcileTombstones(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`UPDATE notes
SET deletion_source=COALESCE(NULLIF(deletion_source, ''), NULLIF(source, ''), 'unknown'),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), ?)
WHERE deleted_at IS NOT NULL`,
		`UPDATE document_panels
SET deletion_source=COALESCE(NULLIF(deletion_source, ''), NULLIF(source, ''), 'unknown'),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), ?)
WHERE deleted_at IS NOT NULL`,
		`UPDATE source_objects
SET deletion_source=COALESCE(NULLIF(deletion_source, ''), NULLIF(source, ''), 'unknown'),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), ?)
WHERE deleted_at IS NOT NULL`,
		`UPDATE transcript_chunks
SET deleted_at=COALESCE(deleted_at, (SELECT deleted_at FROM notes WHERE notes.id = transcript_chunks.document_id)),
    deletion_source=COALESCE(NULLIF(deletion_source, ''), (SELECT deletion_source FROM notes WHERE notes.id = transcript_chunks.document_id)),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), (SELECT deletion_reason FROM notes WHERE notes.id = transcript_chunks.document_id))
WHERE EXISTS (SELECT 1 FROM notes WHERE notes.id = transcript_chunks.document_id AND notes.deleted_at IS NOT NULL)`,
		`UPDATE document_panels
SET deleted_at=COALESCE(deleted_at, (SELECT deleted_at FROM notes WHERE notes.id = document_panels.document_id)),
    deletion_source=COALESCE(NULLIF(deletion_source, ''), (SELECT deletion_source FROM notes WHERE notes.id = document_panels.document_id)),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), (SELECT deletion_reason FROM notes WHERE notes.id = document_panels.document_id))
WHERE EXISTS (SELECT 1 FROM notes WHERE notes.id = document_panels.document_id AND notes.deleted_at IS NOT NULL)`,
		`UPDATE source_objects
SET deleted_at=COALESCE(deleted_at, (SELECT deleted_at FROM notes WHERE notes.id = source_objects.document_id)),
    deletion_source=COALESCE(NULLIF(deletion_source, ''), (SELECT deletion_source FROM notes WHERE notes.id = source_objects.document_id)),
    deletion_reason=COALESCE(NULLIF(deletion_reason, ''), (SELECT deletion_reason FROM notes WHERE notes.id = source_objects.document_id))
WHERE EXISTS (SELECT 1 FROM notes WHERE notes.id = source_objects.document_id AND notes.deleted_at IS NOT NULL)`,
	}
	for index, statement := range statements {
		var args []any
		if index < 3 {
			args = []any{DeletionReasonLegacyField}
		}
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return fmt.Errorf("reconcile tombstones step %d: %w", index+1, err)
		}
	}
	return nil
}

func (s *Store) TombstoneSourceObject(ctx context.Context, source model.Source, kind, sourceID string, deletion Deletion) error {
	if sourceID == "" {
		return nil
	}
	if deletion.At.IsZero() {
		deletion.At = time.Now().UTC()
	}
	_, err := s.DB().ExecContext(ctx, `
UPDATE source_objects
SET deleted_at=COALESCE(deleted_at, ?),
    deletion_source=COALESCE(deletion_source, ?),
    deletion_reason=COALESCE(deletion_reason, ?)
WHERE source = ? AND kind = ? AND source_id = ?`,
		deletion.At.UTC().Format(time.RFC3339Nano), string(deletion.Source), deletion.Reason,
		string(source), kind, sourceID)
	if err != nil {
		return fmt.Errorf("tombstone source object %s/%s/%s: %w", source, kind, sourceID, err)
	}
	return nil
}

// TombstoneDocument retains the canonical note and marks every archived child
// and raw source row for the document. Repeated delete events preserve the
// first recorded tombstone.
func (s *Store) TombstoneDocument(ctx context.Context, documentID string, deletion Deletion) error {
	if documentID == "" {
		return nil
	}
	if deletion.At.IsZero() {
		deletion.At = time.Now().UTC()
	}
	deletedAt := deletion.At.UTC().Format(time.RFC3339Nano)
	tx, err := s.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tombstone: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO notes (
  id, type, created_at, updated_at, deleted_at, source, last_seen_at,
  deletion_source, deletion_reason
) VALUES (?, 'unknown', ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  deleted_at=COALESCE(notes.deleted_at, excluded.deleted_at),
  deletion_source=COALESCE(notes.deletion_source, excluded.deletion_source),
  deletion_reason=COALESCE(notes.deletion_reason, excluded.deletion_reason)`,
		documentID, deletedAt, deletedAt, deletedAt, string(deletion.Source), deletedAt,
		string(deletion.Source), deletion.Reason); err != nil {
		return fmt.Errorf("tombstone note %s: %w", documentID, err)
	}
	for _, table := range []string{"transcript_chunks", "document_panels", "source_objects"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
UPDATE %s
SET deleted_at=COALESCE(deleted_at, ?),
    deletion_source=COALESCE(deletion_source, ?),
    deletion_reason=COALESCE(deletion_reason, ?)
WHERE document_id = ?`, table), deletedAt, string(deletion.Source), deletion.Reason, documentID); err != nil {
			return fmt.Errorf("tombstone %s for %s: %w", table, documentID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tombstone: %w", err)
	}
	committed = true
	return nil
}
