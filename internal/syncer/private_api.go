package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/privateapi"
	"github.com/openclaw/graincrawl/internal/store"
)

func PrivateAPI(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	_, tokens, user, err := granola.ReadSupabase(paths.Supabase)
	if err != nil {
		return Result{}, err
	}
	summary := granola.SummarizeToken(tokens, time.Now())
	if !summary.Present {
		return Result{}, fmt.Errorf("Granola access token not found")
	}
	if summary.Expired {
		return Result{}, fmt.Errorf("Granola access token expired; open Granola or use an explicit refresh flow")
	}
	workspace := user.ActiveWorkspaceID
	if workspace == "" && len(user.WorkspaceIDs) > 0 {
		workspace = user.WorkspaceIDs[0]
	}
	client := privateapi.Client{
		AccessToken:   tokens.AccessToken,
		ClientVersion: cfg.API.ClientVersion,
		Platform:      cfg.API.Platform,
		WorkspaceID:   workspace,
	}
	now := time.Now().UTC()
	for _, workspaceID := range user.WorkspaceIDs {
		if err := retainSourceObject(ctx, st, model.SourcePrivateAPI, "workspace", workspaceID, "", map[string]any{
			"id":     workspaceID,
			"active": workspaceID == workspace,
		}, now); err != nil {
			return Result{}, err
		}
	}
	return syncPrivate(ctx, client, st, opts, cfg.API.IncludeSharedWithMe)
}

func syncPrivate(ctx context.Context, client privateapi.Client, st *store.Store, opts Options, includeShared bool) (Result, error) {
	source := model.SourcePrivateAPI
	started := time.Now().UTC()
	result := Result{Source: source}
	docs, err := client.GetDocuments(ctx, privateapi.DocumentsRequest{IncludeSharedWithMe: includeShared})
	if err != nil {
		return result, err
	}
	all := append([]privateapi.Document{}, docs.Docs...)
	all = append(all, docs.Shared...)
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	if hydrated, err := client.GetDocumentsBatch(ctx, documentIDs(all)); err == nil && len(hydrated.Docs) > 0 {
		all = mergeHydratedDocuments(all, hydrated.Docs)
	}
	now := time.Now().UTC()
	for _, doc := range all {
		if err := retainSourceObject(ctx, st, source, "document", doc.ID, doc.ID, doc, now); err != nil {
			return result, err
		}
		if err := retainPeople(ctx, st, source, doc.ID, doc.People, now); err != nil {
			return result, err
		}
		note, err := privateapi.NoteFromDocument(doc, now)
		if err != nil {
			return result, err
		}
		if err := st.UpsertNote(ctx, note); err != nil {
			return result, err
		}
		result.Notes++
		if opts.IncludeTranscripts {
			chunks, err := client.GetDocumentTranscript(ctx, doc.ID)
			if err == nil {
				for _, chunk := range chunks {
					if err := retainSourceObject(ctx, st, source, "transcript_chunk", chunk.ID, doc.ID, chunk, now); err != nil {
						return result, err
					}
					modelChunk, err := privateapi.TranscriptToModel(chunk)
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
		if opts.IncludePanels {
			panels, err := client.GetDocumentPanels(ctx, doc.ID)
			if err == nil {
				for _, panel := range panels {
					if err := retainSourceObject(ctx, st, source, "panel", panel.ID, doc.ID, panel, now); err != nil {
						return result, err
					}
					modelPanel, err := privateapi.PanelToModel(panel)
					if err != nil {
						return result, err
					}
					if err := st.UpsertPanel(ctx, modelPanel); err != nil {
						return result, err
					}
					result.Panels++
				}
			}
		}
	}
	completed := time.Now().UTC()
	_, _ = st.InsertSyncRun(ctx, model.SyncRun{Source: source, StartedAt: started, CompletedAt: completed, Status: "ok", Notes: result.Notes, Transcripts: result.Transcripts, Panels: result.Panels})
	return result, nil
}

func documentIDs(docs []privateapi.Document) []string {
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc.ID != "" {
			ids = append(ids, doc.ID)
		}
	}
	return ids
}

func mergeHydratedDocuments(base, hydrated []privateapi.Document) []privateapi.Document {
	byID := make(map[string]privateapi.Document, len(hydrated))
	for _, doc := range hydrated {
		if doc.ID != "" {
			byID[doc.ID] = doc
		}
	}
	out := make([]privateapi.Document, 0, len(base))
	for _, doc := range base {
		if full, ok := byID[doc.ID]; ok {
			out = append(out, full)
			continue
		}
		out = append(out, doc)
	}
	return out
}
