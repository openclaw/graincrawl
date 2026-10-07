package store

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	ckstore "github.com/openclaw/crawlkit/store"
	"github.com/openclaw/graincrawl/internal/model"
	"modernc.org/sqlite"
)

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("unicode_lower", 1, unicodeLower)
}

// unicodeLower matches strings.ToLower. SQLite lower folds ASCII only.
func unicodeLower(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch text := args[0].(type) {
	case string:
		return strings.ToLower(text), nil
	case []byte:
		return strings.ToLower(string(text)), nil
	default:
		return nil, fmt.Errorf("unicode_lower expects text")
	}
}

func (s *Store) SearchNotes(ctx context.Context, query string, limit int) ([]model.Note, error) {
	if limit <= 0 {
		limit = 50
	}
	needle := "%" + ckstore.EscapeLike(strings.ToLower(strings.TrimSpace(query))) + "%"
	rows, err := s.DB().QueryContext(ctx, `
SELECT DISTINCT notes.id, notes.title, notes.type, notes.status, notes.created_at, notes.updated_at,
  notes.deleted_at, notes.workspace_id, notes.calendar_event_id, notes.notes_plain,
  notes.notes_markdown, notes.summary_text, notes.summary_markdown, notes.source,
  notes.payload_hash, notes.last_seen_at, notes.deletion_source, notes.deletion_reason
FROM notes
LEFT JOIN transcript_chunks ON transcript_chunks.document_id = notes.id
LEFT JOIN document_panels ON document_panels.document_id = notes.id
WHERE unicode_lower(coalesce(notes.title, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(notes.notes_plain, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(notes.notes_markdown, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(notes.summary_text, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(notes.summary_markdown, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(transcript_chunks.text, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(document_panels.content_plain, '')) LIKE ? ESCAPE '\'
   OR unicode_lower(coalesce(document_panels.content_markdown, '')) LIKE ? ESCAPE '\'
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
