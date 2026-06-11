package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/cachev6"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func DesktopCache(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	encryptedOnlyMessage := ""
	if granola.EncryptedOnlyState(paths) {
		encryptedOnlyMessage = granola.EncryptedOnlyStateMessage
	}
	if granola.EncryptedCacheState(paths) {
		result := Result{Source: model.SourceDesktopCache, Message: encryptedOnlyMessage}
		return result, fmt.Errorf("desktop-cache source requires plaintext cache-v6.json: %s", encryptedOnlyMessage)
	}
	file, err := cachev6.Read(paths.CacheV6)
	if err != nil {
		result := Result{Source: model.SourceDesktopCache, Message: encryptedOnlyMessage}
		if encryptedOnlyMessage != "" {
			return result, fmt.Errorf("%w: %s", err, encryptedOnlyMessage)
		}
		return result, err
	}
	return importDesktopCache(ctx, st, opts, file, model.SourceDesktopCache, encryptedOnlyMessage)
}

func encryptedDesktopCache(ctx context.Context, st *store.Store, opts Options, raw []byte) (Result, error) {
	file, err := cachev6.Parse(raw)
	if err != nil {
		return Result{Source: model.SourceEncryptedJSON}, err
	}
	return importDesktopCache(ctx, st, opts, file, model.SourceEncryptedJSON, "unlocked encrypted cache-v6.json in memory")
}

func importDesktopCache(ctx context.Context, st *store.Store, opts Options, file cachev6.File, source model.Source, message string) (Result, error) {
	started := time.Now().UTC()
	result := Result{Source: source, Message: message}
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
		note.Source = source
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
	_, _ = st.InsertSyncRun(ctx, model.SyncRun{Source: source, StartedAt: started, CompletedAt: completed, Status: "ok", Notes: result.Notes, Transcripts: result.Transcripts, Panels: result.Panels, Message: result.Message})
	return result, nil
}
