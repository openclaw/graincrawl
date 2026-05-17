package store

import (
	"context"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func (s *Store) InsertSyncRun(ctx context.Context, run model.SyncRun) (int64, error) {
	res, err := s.DB().ExecContext(ctx, `
INSERT INTO sync_runs (source, started_at, completed_at, status, notes, transcripts, panels, message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(run.Source), run.StartedAt.Format(time.RFC3339Nano), run.CompletedAt.Format(time.RFC3339Nano),
		run.Status, run.Notes, run.Transcripts, run.Panels, run.Message)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListSyncRuns(ctx context.Context, limit int) ([]model.SyncRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.DB().QueryContext(ctx, `
SELECT id, source, started_at, completed_at, status, notes, transcripts, panels, COALESCE(message, '')
FROM sync_runs
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []model.SyncRun
	for rows.Next() {
		var run model.SyncRun
		var source, started, completed string
		if err := rows.Scan(&run.ID, &source, &started, &completed, &run.Status, &run.Notes, &run.Transcripts, &run.Panels, &run.Message); err != nil {
			return nil, err
		}
		run.Source = model.Source(source)
		run.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
		run.CompletedAt, _ = time.Parse(time.RFC3339Nano, completed)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
