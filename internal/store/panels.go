package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) UpsertPanel(ctx context.Context, panel model.Panel) error {
	_, err := s.DB().ExecContext(ctx, `
INSERT INTO document_panels (
  id, document_id, title, template_slug, content_plain, content_markdown,
  content_json, created_at, updated_at, last_viewed_at, ydoc_version,
  ydoc_cached_at, source
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  document_id=excluded.document_id,
  title=excluded.title,
  template_slug=excluded.template_slug,
  content_plain=excluded.content_plain,
  content_markdown=excluded.content_markdown,
  content_json=excluded.content_json,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  last_viewed_at=excluded.last_viewed_at,
  ydoc_version=excluded.ydoc_version,
  ydoc_cached_at=excluded.ydoc_cached_at,
  source=excluded.source`,
		panel.ID, panel.DocumentID, panel.Title, panel.TemplateSlug, panel.ContentPlain,
		panel.ContentMarkdown, panel.ContentJSON, panel.CreatedAt.Format(time.RFC3339Nano),
		timePtr(panel.UpdatedAt), timePtr(panel.LastViewedAt), panel.YdocVersion,
		timePtr(panel.YdocCachedAt), string(panel.Source))
	return err
}

func (s *Store) ListPanels(ctx context.Context, documentID string) ([]model.Panel, error) {
	rows, err := s.DB().QueryContext(ctx, `
SELECT id, document_id, title, template_slug, content_plain, content_markdown,
  content_json, created_at, updated_at, last_viewed_at, ydoc_version,
  ydoc_cached_at, source
FROM document_panels
WHERE document_id = ?
ORDER BY created_at DESC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var panels []model.Panel
	for rows.Next() {
		panel, err := scanPanel(rows)
		if err != nil {
			return nil, err
		}
		panels = append(panels, panel)
	}
	return panels, rows.Err()
}

func scanPanel(rows *sql.Rows) (model.Panel, error) {
	var p model.Panel
	var title, slug, plain, markdown, updated, viewed, cached, source sql.NullString
	var ydoc sql.NullInt64
	var created string
	if err := rows.Scan(&p.ID, &p.DocumentID, &title, &slug, &plain, &markdown, &p.ContentJSON, &created, &updated, &viewed, &ydoc, &cached, &source); err != nil {
		return model.Panel{}, err
	}
	p.Title = stringPtr(title)
	p.TemplateSlug = stringPtr(slug)
	p.ContentPlain = stringPtr(plain)
	p.ContentMarkdown = stringPtr(markdown)
	p.Source = model.Source(source.String)
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	p.UpdatedAt = parseNullableTime(updated)
	p.LastViewedAt = parseNullableTime(viewed)
	p.YdocCachedAt = parseNullableTime(cached)
	if ydoc.Valid {
		p.YdocVersion = &ydoc.Int64
	}
	return p, nil
}

func parseNullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &t
}
