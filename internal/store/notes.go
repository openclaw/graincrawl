package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) UpsertNote(ctx context.Context, note model.Note) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO notes (
  id, title, type, status, created_at, updated_at, deleted_at, workspace_id,
  calendar_event_id, notes_plain, notes_markdown, summary_text,
  summary_markdown, source, payload_hash, last_seen_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  type=excluded.type,
  status=excluded.status,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  deleted_at=excluded.deleted_at,
  workspace_id=excluded.workspace_id,
  calendar_event_id=excluded.calendar_event_id,
  notes_plain=excluded.notes_plain,
  notes_markdown=excluded.notes_markdown,
  summary_text=excluded.summary_text,
  summary_markdown=excluded.summary_markdown,
  source=excluded.source,
  payload_hash=excluded.payload_hash,
  last_seen_at=excluded.last_seen_at`,
		note.ID, note.Title, note.Type, note.Status, note.CreatedAt.Format(time.RFC3339Nano),
		note.UpdatedAt.Format(time.RFC3339Nano), timePtr(note.DeletedAt), note.WorkspaceID,
		note.CalendarEventID, note.NotesPlain, note.NotesMarkdown, note.SummaryText,
		note.SummaryMarkdown, string(note.Source), note.PayloadHash, note.LastSeenAt.Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListNotes(ctx context.Context, limit int) ([]model.Note, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB().QueryContext(ctx, `
SELECT id, title, type, status, created_at, updated_at, deleted_at, workspace_id,
  calendar_event_id, notes_plain, notes_markdown, summary_text,
  summary_markdown, source, payload_hash, last_seen_at
FROM notes
ORDER BY created_at DESC
LIMIT ?`, limit)
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

func (s *Store) GetNote(ctx context.Context, id string) (model.Note, bool, error) {
	row := s.DB().QueryRowContext(ctx, `
SELECT id, title, type, status, created_at, updated_at, deleted_at, workspace_id,
  calendar_event_id, notes_plain, notes_markdown, summary_text,
  summary_markdown, source, payload_hash, last_seen_at
FROM notes
WHERE id = ?`, id)
	note, err := scanNote(row)
	if err == sql.ErrNoRows {
		return model.Note{}, false, nil
	}
	if err != nil {
		return model.Note{}, false, err
	}
	return note, true, nil
}

type noteScanner interface {
	Scan(dest ...any) error
}

func scanNote(scanner noteScanner) (model.Note, error) {
	var n model.Note
	var title, status, deleted, workspace, eventID, plain, markdown, summaryText, summaryMarkdown, source, payloadHash sql.NullString
	var created, updated, seen string
	if err := scanner.Scan(&n.ID, &title, &n.Type, &status, &created, &updated, &deleted, &workspace, &eventID, &plain, &markdown, &summaryText, &summaryMarkdown, &source, &payloadHash, &seen); err != nil {
		return model.Note{}, err
	}
	n.Title = stringPtr(title)
	n.Status = stringPtr(status)
	n.WorkspaceID = stringPtr(workspace)
	n.CalendarEventID = stringPtr(eventID)
	n.NotesPlain = stringPtr(plain)
	n.NotesMarkdown = stringPtr(markdown)
	n.SummaryText = stringPtr(summaryText)
	n.SummaryMarkdown = stringPtr(summaryMarkdown)
	n.Source = model.Source(source.String)
	n.PayloadHash = payloadHash.String
	n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	n.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	n.LastSeenAt, _ = time.Parse(time.RFC3339Nano, seen)
	if deleted.Valid {
		t, _ := time.Parse(time.RFC3339Nano, deleted.String)
		n.DeletedAt = &t
	}
	return n, nil
}
