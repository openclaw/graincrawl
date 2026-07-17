package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

type SourceObject struct {
	Source         model.Source `json:"source"`
	Kind           string       `json:"kind"`
	SourceID       string       `json:"source_id"`
	DocumentID     string       `json:"document_id,omitempty"`
	PayloadJSON    string       `json:"payload_json"`
	PayloadHash    string       `json:"payload_hash"`
	ObservedAt     time.Time    `json:"observed_at"`
	DeletedAt      *time.Time   `json:"deleted_at,omitempty"`
	DeletionSource string       `json:"deletion_source,omitempty"`
	DeletionReason string       `json:"deletion_reason,omitempty"`
}

func (s *Store) UpsertSourceObject(ctx context.Context, obj SourceObject) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO source_objects (source, kind, source_id, document_id, payload_json, payload_hash, observed_at, deleted_at, deletion_source, deletion_reason)
VALUES (?, ?, ?, ?, ?, ?, ?,
  COALESCE(?, (SELECT deleted_at FROM notes WHERE id = ?)),
  COALESCE(?, (SELECT deletion_source FROM notes WHERE id = ?)),
  COALESCE(?, (SELECT deletion_reason FROM notes WHERE id = ?)))
ON CONFLICT(source, kind, source_id) DO UPDATE SET
  document_id=excluded.document_id,
  payload_json=excluded.payload_json,
  payload_hash=excluded.payload_hash,
  observed_at=excluded.observed_at,
  deleted_at=COALESCE(source_objects.deleted_at, excluded.deleted_at),
  deletion_source=COALESCE(source_objects.deletion_source, excluded.deletion_source),
  deletion_reason=COALESCE(source_objects.deletion_reason, excluded.deletion_reason)`,
		string(obj.Source), obj.Kind, obj.SourceID, nullableString(obj.DocumentID),
		obj.PayloadJSON, obj.PayloadHash, obj.ObservedAt.Format(time.RFC3339Nano),
		timePtr(obj.DeletedAt), obj.DocumentID,
		nullableString(obj.DeletionSource), obj.DocumentID,
		nullableString(obj.DeletionReason), obj.DocumentID)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) ListSourceObjects(ctx context.Context, kind string, limit int) ([]SourceObject, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB().QueryContext(ctx, `
SELECT source, kind, source_id, COALESCE(document_id, ''), payload_json, payload_hash, observed_at,
  deleted_at, deletion_source, deletion_reason
FROM source_objects
WHERE (? = '' OR kind = ?)
ORDER BY observed_at DESC
LIMIT ?`, kind, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects []SourceObject
	for rows.Next() {
		obj, err := scanSourceObject(rows)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, rows.Err()
}

type sourceObjectScanner interface {
	Scan(dest ...any) error
}

func scanSourceObject(scanner sourceObjectScanner) (SourceObject, error) {
	var obj SourceObject
	var source, observed string
	var deleted, deletionSource, deletionReason sql.NullString
	if err := scanner.Scan(&source, &obj.Kind, &obj.SourceID, &obj.DocumentID, &obj.PayloadJSON, &obj.PayloadHash, &observed, &deleted, &deletionSource, &deletionReason); err != nil {
		if err == sql.ErrNoRows {
			return SourceObject{}, err
		}
		return SourceObject{}, err
	}
	obj.Source = model.Source(source)
	obj.ObservedAt, _ = time.Parse(time.RFC3339Nano, observed)
	obj.DeletedAt = parseNullableTime(deleted)
	obj.DeletionSource = deletionSource.String
	obj.DeletionReason = deletionReason.String
	return obj, nil
}
