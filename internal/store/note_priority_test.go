package store

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func TestNoteDeletionStubHydration(t *testing.T) {
	for _, source := range []model.Source{model.SourceDesktopCache, model.SourceEncryptedJSON, model.SourcePrivateAPI} {
		t.Run(string(source), func(t *testing.T) {
			ctx := context.Background()
			st, err := Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			deleted := time.Date(2026, 9, 8, 12, 0, 0, 123, time.UTC)
			if err := st.TombstoneDocument(ctx, "note", Deletion{
				At: deleted, Source: model.SourcePrivateAPI, Reason: DeletionReasonExplicitFeed,
			}); err != nil {
				t.Fatal(err)
			}
			if err := st.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
				ID: "chunk", DocumentID: "note", StartTimestamp: deleted,
				EndTimestamp: deleted.Add(time.Second), Source: "mic", Text: "retained child",
			}); err != nil {
				t.Fatal(err)
			}
			if err := st.UpsertPanel(ctx, model.Panel{
				ID: "panel", DocumentID: "note", CreatedAt: deleted, Source: model.SourcePrivateAPI,
			}); err != nil {
				t.Fatal(err)
			}
			if err := st.UpsertSourceObject(ctx, SourceObject{
				Source: model.SourcePrivateAPI, Kind: "document", SourceID: "raw",
				DocumentID: "note", PayloadJSON: `{}`, PayloadHash: "hash", ObservedAt: deleted,
			}); err != nil {
				t.Fatal(err)
			}
			assertTombstones := func() {
				t.Helper()
				for _, table := range []string{"notes", "transcript_chunks", "document_panels", "source_objects"} {
					var count int
					query := fmt.Sprintf(`SELECT count(*) FROM %s
						WHERE deleted_at = ? AND deletion_source = ? AND deletion_reason = ?`, table)
					if err := st.DB().QueryRowContext(ctx, query, deleted.Format(time.RFC3339Nano),
						string(model.SourcePrivateAPI), DeletionReasonExplicitFeed).Scan(&count); err != nil {
						t.Fatal(err)
					}
					if count != 1 {
						t.Fatalf("%s lost its original tombstone", table)
					}
				}
			}
			assertTombstones()
			body, summary := "recovered content", "recovered summary"
			note := model.Note{
				ID: "note", Type: "meeting", Source: source, PayloadHash: "content-hash",
				CreatedAt: deleted.Add(-2 * time.Hour), UpdatedAt: deleted.Add(-time.Hour),
				LastSeenAt: deleted.Add(time.Hour), NotesPlain: &body, SummaryText: &summary,
			}
			if err := st.UpsertNote(ctx, note); err != nil {
				t.Fatal(err)
			}
			got, found, err := st.GetNote(ctx, note.ID)
			if err != nil || !found {
				t.Fatalf("hydrated note: found=%t err=%v", found, err)
			}
			want := note
			want.DeletedAt, want.DeletionSource, want.DeletionReason = &deleted, string(model.SourcePrivateAPI), DeletionReasonExplicitFeed
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("hydration changed content or deletion provenance: got=%+v want=%+v", got, want)
			}
			assertTombstones()

			// Once populated, normal content freshness and source precedence apply.
			for _, nextSource := range []model.Source{source, model.SourceDesktopCache} {
				incoming := note
				incoming.Source = nextSource
				incoming.UpdatedAt = note.UpdatedAt.Add(-time.Hour)
				if source == model.SourcePrivateAPI && nextSource == model.SourceDesktopCache {
					incoming.UpdatedAt = deleted.Add(2 * time.Hour)
				}
				replacement := "must not replace recovered content"
				incoming.NotesPlain = &replacement
				if err := st.UpsertNote(ctx, incoming); err != nil {
					t.Fatal(err)
				}
				after, _, err := st.GetNote(ctx, note.ID)
				if err != nil || !reflect.DeepEqual(after, want) {
					t.Fatalf("populated note lost precedence: %+v err=%v", after, err)
				}
				assertTombstones()
			}
		})
	}
}

func TestNotePrecedenceDoesNotTreatEmptyContentAsDeletionStub(t *testing.T) {
	for _, change := range []string{
		"title = ''", "status = ''", "workspace_id = ''", "calendar_event_id = ''",
		"notes_plain = ''", "notes_markdown = ''", "summary_text = ''", "summary_markdown = ''",
		"payload_hash = ''", "type = 'meeting'", "updated_at = '2026-09-08T11:00:00Z'",
		"created_at = '2026-09-08T11:00:00Z'", "last_seen_at = '2026-09-08T13:00:00Z'",
		"deletion_source = 'other'", "deletion_reason = NULL", "deleted_at = NULL",
	} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			st, err := Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
			if err := st.TombstoneDocument(ctx, "note", Deletion{
				At: now, Source: model.SourcePrivateAPI, Reason: DeletionReasonExplicitFeed,
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := st.DB().ExecContext(ctx, "UPDATE notes SET "+change+" WHERE id = 'note'"); err != nil {
				t.Fatal(err)
			}
			before, _, err := st.GetNote(ctx, "note")
			if err != nil {
				t.Fatal(err)
			}
			body := "fallback"
			if err := st.UpsertNote(ctx, model.Note{
				ID: "note", Type: "meeting", Source: model.SourceDesktopCache,
				CreatedAt: now, UpdatedAt: now.Add(time.Hour), LastSeenAt: now.Add(time.Hour), NotesPlain: &body,
			}); err != nil {
				t.Fatal(err)
			}
			after, _, err := st.GetNote(ctx, "note")
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("non-stub lost precedence: %+v err=%v", after, err)
			}
		})
	}
}

func TestNotePriorityAndFreshness(t *testing.T) {
	for _, tc := range []struct {
		name               string
		from, to           model.Source
		older, keep, clear bool
	}{
		{"older cache", model.SourcePrivateAPI, model.SourceDesktopCache, true, true, true},
		{"newer cache", model.SourcePrivateAPI, model.SourceDesktopCache, false, true, true},
		{"encrypted cache", model.SourcePrivateAPI, model.SourceEncryptedJSON, false, true, false},
		{"older API", model.SourcePrivateAPI, model.SourcePrivateAPI, true, true, false},
		{"API clears", model.SourcePrivateAPI, model.SourcePrivateAPI, false, false, true},
		{"cache clears", model.SourceDesktopCache, model.SourceDesktopCache, false, false, true},
		{"equivalent cache", model.SourceDesktopCache, model.SourceEncryptedJSON, true, true, false},
		{"public source freshness", model.SourcePublicAPI, model.SourcePublicAPI, true, true, false},
		{"public cache compatibility", model.SourcePublicAPI, model.SourceDesktopCache, true, false, false},
		{"API promotion", model.SourceDesktopCache, model.SourcePrivateAPI, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			st, err := Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			now := time.Date(2026, 9, 8, 12, 0, 0, 500, time.UTC)
			body, summary := "original", "summary"
			original := model.Note{ID: "note", Type: "meeting", CreatedAt: now, UpdatedAt: now, LastSeenAt: now, NotesPlain: &body, SummaryText: &summary, Source: tc.from}
			if err := st.UpsertNote(ctx, original); err != nil {
				t.Fatal(err)
			}
			incoming := original
			incoming.Source = tc.to
			incoming.UpdatedAt = now.Add(time.Second)
			if tc.older {
				incoming.UpdatedAt = now.Add(-500 * time.Nanosecond)
			}
			replacement := "replacement"
			incoming.NotesPlain = &replacement
			if tc.clear {
				incoming.NotesPlain, incoming.SummaryText = nil, nil
			}
			deleted := now.Add(time.Hour)
			incoming.DeletedAt, incoming.DeletionSource = &deleted, string(tc.to)
			if err := st.UpsertNote(ctx, incoming); err != nil {
				t.Fatal(err)
			}
			got, ok, err := st.GetNote(ctx, "note")
			if err != nil || !ok {
				t.Fatalf("read: %v %v", ok, err)
			}
			if tc.keep {
				if got.Source != original.Source || !got.UpdatedAt.Equal(original.UpdatedAt) || got.NotesPlain == nil || *got.NotesPlain != body || got.SummaryText == nil || *got.SummaryText != summary {
					t.Fatalf("canonical content regressed: %+v", got)
				}
			} else if got.Source != incoming.Source || !got.UpdatedAt.Equal(incoming.UpdatedAt) || (tc.clear && (got.NotesPlain != nil || got.SummaryText != nil)) {
				t.Fatalf("authoritative update rejected: %+v", got)
			}
			if got.DeletedAt == nil || !got.DeletedAt.Equal(deleted) {
				t.Fatal("lost tombstone")
			}
			incoming.DeletedAt = nil
			if err := st.UpsertNote(ctx, incoming); err != nil {
				t.Fatal(err)
			}
			got, _, err = st.GetNote(ctx, "note")
			if err != nil || got.DeletedAt == nil {
				t.Fatalf("cleared existing tombstone: %v", err)
			}
		})
	}
}
