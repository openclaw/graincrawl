package store

import (
	"context"
	"time"

	ckstore "github.com/openclaw/crawlkit/store"
)

type Status struct {
	DBPath      string    `json:"db_path"`
	Notes       int64     `json:"notes"`
	Transcripts int64     `json:"transcripts"`
	Panels      int64     `json:"panels"`
	Sources     int64     `json:"sources"`
	SyncRuns    int64     `json:"sync_runs"`
	LastSyncAt  time.Time `json:"last_sync_at,omitempty"`
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	status := Status{DBPath: s.Path()}
	counts := map[string]*int64{
		"notes":             &status.Notes,
		"transcript_chunks": &status.Transcripts,
		"document_panels":   &status.Panels,
		"source_objects":    &status.Sources,
		"sync_runs":         &status.SyncRuns,
	}
	for table, target := range counts {
		if err := s.DB().QueryRowContext(ctx, "select count(*) from "+ckstore.QuoteIdent(table)).Scan(target); err != nil {
			return Status{}, err
		}
	}
	var last string
	if err := s.DB().QueryRowContext(ctx, `select coalesce(max(completed_at), '') from sync_runs where status = 'ok'`).Scan(&last); err != nil {
		return Status{}, err
	}
	if last != "" {
		status.LastSyncAt, _ = time.Parse(time.RFC3339Nano, last)
	}
	return status, nil
}
