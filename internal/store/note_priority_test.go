package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

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
