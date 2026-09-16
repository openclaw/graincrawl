package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func completeSync(ctx context.Context, st *store.Store, started time.Time, result Result) (Result, error) {
	_, err := st.InsertSyncRun(ctx, model.SyncRun{
		Source:      result.Source,
		StartedAt:   started,
		CompletedAt: time.Now().UTC(),
		Status:      "ok",
		Notes:       result.Notes,
		Transcripts: result.Transcripts,
		Panels:      result.Panels,
		Message:     result.Message,
	})
	if err != nil {
		return result, fmt.Errorf("record completed sync: %w", err)
	}
	return result, nil
}
