package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) UpsertTranscriptChunk(ctx context.Context, chunk model.TranscriptChunk) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO transcript_chunks (
  id, document_id, start_timestamp, end_timestamp, source, is_final,
  transcriber_user_id, text, payload_hash, deleted_at, deletion_source, deletion_reason
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,
  COALESCE(?, (SELECT deleted_at FROM notes WHERE id = ?)),
  COALESCE(?, (SELECT deletion_source FROM notes WHERE id = ?)),
  COALESCE(?, (SELECT deletion_reason FROM notes WHERE id = ?)))
ON CONFLICT(id) DO UPDATE SET
  document_id=excluded.document_id,
  start_timestamp=excluded.start_timestamp,
  end_timestamp=excluded.end_timestamp,
  source=excluded.source,
  is_final=excluded.is_final,
  transcriber_user_id=excluded.transcriber_user_id,
  text=excluded.text,
  payload_hash=excluded.payload_hash,
  deleted_at=COALESCE(transcript_chunks.deleted_at, excluded.deleted_at),
  deletion_source=COALESCE(transcript_chunks.deletion_source, excluded.deletion_source),
  deletion_reason=COALESCE(transcript_chunks.deletion_reason, excluded.deletion_reason)`,
		chunk.ID, chunk.DocumentID, chunk.StartTimestamp.Format(time.RFC3339Nano),
		chunk.EndTimestamp.Format(time.RFC3339Nano), chunk.Source, boolInt(chunk.IsFinal),
		chunk.TranscriberUserID, chunk.Text, chunk.PayloadHash,
		timePtr(chunk.DeletedAt), chunk.DocumentID,
		nullableString(chunk.DeletionSource), chunk.DocumentID,
		nullableString(chunk.DeletionReason), chunk.DocumentID)
	return err
}

func (s *Store) ListTranscript(ctx context.Context, documentID string) ([]model.TranscriptChunk, error) {
	rows, err := s.DB().QueryContext(ctx, `
SELECT id, document_id, start_timestamp, end_timestamp, source, is_final,
  transcriber_user_id, text, payload_hash
  , deleted_at, deletion_source, deletion_reason
FROM transcript_chunks
WHERE document_id = ?
ORDER BY start_timestamp ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []model.TranscriptChunk
	for rows.Next() {
		var chunk model.TranscriptChunk
		var start, end string
		var deleted, deletionSource, deletionReason sql.NullString
		var final int
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &start, &end, &chunk.Source, &final, &chunk.TranscriberUserID, &chunk.Text, &chunk.PayloadHash, &deleted, &deletionSource, &deletionReason); err != nil {
			return nil, err
		}
		chunk.StartTimestamp, _ = time.Parse(time.RFC3339Nano, start)
		chunk.EndTimestamp, _ = time.Parse(time.RFC3339Nano, end)
		chunk.IsFinal = final != 0
		chunk.DeletedAt = parseNullableTime(deleted)
		chunk.DeletionSource = deletionSource.String
		chunk.DeletionReason = deletionReason.String
		chunks = append(chunks, chunk)
	}
	return chunks, rows.Err()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
