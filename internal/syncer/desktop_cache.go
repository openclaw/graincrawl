package syncer

import (
	"context"
	"time"

	"github.com/openclaw/graincrawl/internal/cachev6"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func DesktopCache(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	source := model.SourceDesktopCache
	started := time.Now().UTC()
	result := Result{Source: source}
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	file, err := cachev6.Read(paths.CacheV6)
	if err != nil {
		return result, err
	}
	now := time.Now().UTC()
	count := 0
	for _, doc := range file.Cache.State.Documents {
		if opts.Limit > 0 && count >= opts.Limit {
			break
		}
		if err := retainSourceObject(ctx, st, source, "document", doc.ID, doc.ID, doc, now); err != nil {
			return result, err
		}
		if err := retainPeople(ctx, st, source, doc.ID, doc.People, now); err != nil {
			return result, err
		}
		note, err := cachev6.NoteFromDocument(doc, now)
		if err != nil {
			return result, err
		}
		if err := st.UpsertNote(ctx, note); err != nil {
			return result, err
		}
		count++
		result.Notes++
		if opts.IncludeTranscripts {
			for _, chunk := range file.Cache.State.Transcripts[doc.ID] {
				if err := retainSourceObject(ctx, st, source, "transcript_chunk", chunk.ID, doc.ID, chunk, now); err != nil {
					return result, err
				}
				modelChunk, err := cachev6.TranscriptFromCache(chunk)
				if err != nil {
					return result, err
				}
				if err := st.UpsertTranscriptChunk(ctx, modelChunk); err != nil {
					return result, err
				}
				result.Transcripts++
			}
		}
	}
	completed := time.Now().UTC()
	_, _ = st.InsertSyncRun(ctx, model.SyncRun{Source: source, StartedAt: started, CompletedAt: completed, Status: "ok", Notes: result.Notes, Transcripts: result.Transcripts, Panels: result.Panels})
	return result, nil
}
