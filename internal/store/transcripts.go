package store

import (
	"context"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) UpsertTranscriptChunk(ctx context.Context, chunk model.TranscriptChunk) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO transcript_chunks (
  id, document_id, start_timestamp, end_timestamp, source, is_final,
  transcriber_user_id, text, payload_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  document_id=excluded.document_id,
  start_timestamp=excluded.start_timestamp,
  end_timestamp=excluded.end_timestamp,
  source=excluded.source,
  is_final=excluded.is_final,
  transcriber_user_id=excluded.transcriber_user_id,
  text=excluded.text,
  payload_hash=excluded.payload_hash`,
		chunk.ID, chunk.DocumentID, chunk.StartTimestamp.Format(time.RFC3339Nano),
		chunk.EndTimestamp.Format(time.RFC3339Nano), chunk.Source, boolInt(chunk.IsFinal),
		chunk.TranscriberUserID, chunk.Text, chunk.PayloadHash)
	return err
}

func (s *Store) ListTranscript(ctx context.Context, documentID string) ([]model.TranscriptChunk, error) {
	rows, err := s.DB().QueryContext(ctx, `
SELECT id, document_id, start_timestamp, end_timestamp, source, is_final,
  transcriber_user_id, text, payload_hash
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
		var final int
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &start, &end, &chunk.Source, &final, &chunk.TranscriberUserID, &chunk.Text, &chunk.PayloadHash); err != nil {
			return nil, err
		}
		chunk.StartTimestamp, _ = time.Parse(time.RFC3339Nano, start)
		chunk.EndTimestamp, _ = time.Parse(time.RFC3339Nano, end)
		chunk.IsFinal = final != 0
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
