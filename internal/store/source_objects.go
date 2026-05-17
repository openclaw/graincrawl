package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

type SourceObject struct {
	Source      model.Source `json:"source"`
	Kind        string       `json:"kind"`
	SourceID    string       `json:"source_id"`
	DocumentID  string       `json:"document_id,omitempty"`
	PayloadJSON string       `json:"payload_json"`
	PayloadHash string       `json:"payload_hash"`
	ObservedAt  time.Time    `json:"observed_at"`
}

func (s *Store) UpsertSourceObject(ctx context.Context, obj SourceObject) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO source_objects (source, kind, source_id, document_id, payload_json, payload_hash, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(source, kind, source_id) DO UPDATE SET
  document_id=excluded.document_id,
  payload_json=excluded.payload_json,
  payload_hash=excluded.payload_hash,
  observed_at=excluded.observed_at`,
		string(obj.Source), obj.Kind, obj.SourceID, nullableString(obj.DocumentID),
		obj.PayloadJSON, obj.PayloadHash, obj.ObservedAt.Format(time.RFC3339Nano))
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
SELECT source, kind, source_id, COALESCE(document_id, ''), payload_json, payload_hash, observed_at
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
	if err := scanner.Scan(&source, &obj.Kind, &obj.SourceID, &obj.DocumentID, &obj.PayloadJSON, &obj.PayloadHash, &observed); err != nil {
		if err == sql.ErrNoRows {
			return SourceObject{}, err
		}
		return SourceObject{}, err
	}
	obj.Source = model.Source(source)
	obj.ObservedAt, _ = time.Parse(time.RFC3339Nano, observed)
	return obj, nil
}
