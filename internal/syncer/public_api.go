package syncer

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/publicapi"
	"github.com/openclaw/graincrawl/internal/store"
)

var ErrPublicAPIKeyNotFound = errors.New("Granola public API key not found in " + publicapi.KeyEnv)

func PublicAPI(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	key := os.Getenv(publicapi.KeyEnv)
	if key == "" {
		return Result{Source: model.SourcePublicAPI}, ErrPublicAPIKeyNotFound
	}
	client := &publicapi.Client{
		APIKey:  key,
		BaseURL: cfg.API.PublicBaseURL,
	}
	return syncPublic(ctx, client, st, opts)
}

func syncPublic(ctx context.Context, client *publicapi.Client, st *store.Store, opts Options) (Result, error) {
	source := model.SourcePublicAPI
	started := time.Now().UTC()
	result := Result{Source: source}
	if opts.IncludePanels {
		result.Message = "official public API does not expose panels or deletion events"
	} else {
		result.Message = "official public API does not expose deletion events"
	}

	cursor := ""
	for opts.Limit <= 0 || result.Notes < opts.Limit {
		pageSize := 30
		if remaining := opts.Limit - result.Notes; opts.Limit > 0 && remaining < pageSize {
			pageSize = remaining
		}
		page, err := client.ListNotes(ctx, cursor, pageSize)
		if err != nil {
			return result, err
		}
		for _, summary := range page.Notes {
			note, err := client.GetNote(ctx, summary.ID, opts.IncludeTranscripts)
			if err != nil {
				return result, err
			}
			now := time.Now().UTC()
			if err := retainSourceObject(ctx, st, source, "document", note.ID, note.ID, note, now); err != nil {
				return result, err
			}
			modelNote, err := publicapi.NoteToModel(note, now)
			if err != nil {
				return result, err
			}
			if err := st.UpsertNote(ctx, modelNote); err != nil {
				return result, err
			}
			result.Notes++

			if opts.IncludeTranscripts {
				occurrences := make(map[string]int)
				for _, transcript := range note.Transcript {
					identityHash := publicapi.TranscriptIdentityHash(transcript)
					occurrence := occurrences[identityHash]
					occurrences[identityHash] = occurrence + 1
					modelChunk, err := publicapi.TranscriptToModel(note.ID, transcript, occurrence)
					if err != nil {
						return result, err
					}
					if err := retainSourceObject(ctx, st, source, "transcript_chunk", modelChunk.ID, note.ID, transcript, now); err != nil {
						return result, err
					}
					if err := st.UpsertTranscriptChunk(ctx, modelChunk); err != nil {
						return result, err
					}
					result.Transcripts++
				}
			}
			if opts.Limit > 0 && result.Notes >= opts.Limit {
				break
			}
		}
		if !page.HasMore {
			break
		}
		if page.Cursor == nil || *page.Cursor == "" || *page.Cursor == cursor {
			return result, errors.New("granola public API returned an invalid pagination cursor")
		}
		cursor = *page.Cursor
	}

	completed := time.Now().UTC()
	_, _ = st.InsertSyncRun(ctx, model.SyncRun{
		Source:      source,
		StartedAt:   started,
		CompletedAt: completed,
		Status:      "ok",
		Notes:       result.Notes,
		Transcripts: result.Transcripts,
		Message:     result.Message,
	})
	return result, nil
}
