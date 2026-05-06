package store

import (
	"context"
	"time"

	"github.com/vincentkoc/graincrawl/internal/model"
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
