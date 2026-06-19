package store

import (
	"context"
	"strings"

	ckstore "github.com/openclaw/crawlkit/store"
	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) SearchNotes(ctx context.Context, query string, limit int) ([]model.Note, error) {
	if limit <= 0 {
		limit = 50
	}
	needle := "%" + ckstore.EscapeLike(strings.ToLower(strings.TrimSpace(query))) + "%"
	rows, err := s.DB().QueryContext(ctx, `
SELECT DISTINCT notes.id, notes.title, notes.type, notes.status, notes.created_at, notes.updated_at,
  notes.deleted_at, notes.workspace_id, notes.calendar_event_id, notes.notes_plain,
  notes.notes_markdown, notes.summary_text, notes.summary_markdown, notes.source,
  notes.payload_hash, notes.last_seen_at
FROM notes
LEFT JOIN transcript_chunks ON transcript_chunks.document_id = notes.id
LEFT JOIN document_panels ON document_panels.document_id = notes.id
WHERE lower(coalesce(notes.title, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(notes.notes_plain, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(notes.notes_markdown, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(notes.summary_text, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(notes.summary_markdown, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(transcript_chunks.text, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(document_panels.content_plain, '')) LIKE ? ESCAPE '\'
   OR lower(coalesce(document_panels.content_markdown, '')) LIKE ? ESCAPE '\'
ORDER BY notes.updated_at DESC
LIMIT ?`, needle, needle, needle, needle, needle, needle, needle, needle, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []model.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}
