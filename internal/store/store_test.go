package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func TestStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graincrawl.db")
	st, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC().Round(0)
	title := "Planning"
	note := model.Note{
		ID:         "note-1",
		Title:      &title,
		Type:       "meeting",
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     model.SourcePrivateAPI,
		LastSeenAt: now,
	}
	if err := st.UpsertNote(ctx, note); err != nil {
		t.Fatal(err)
	}
	percentTitle := "100% Ready"
	percentNote := note
	percentNote.ID = "note-2"
	percentNote.Title = &percentTitle
	if err := st.UpsertNote(ctx, percentNote); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GetNote(ctx, "note-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.ID != note.ID || got.Title == nil || *got.Title != title {
		t.Fatalf("unexpected note: %#v", got)
	}
	results, err := st.SearchNotes(ctx, "planning", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != note.ID {
		t.Fatalf("unexpected search results: %#v", results)
	}
	literalPercent, err := st.SearchNotes(ctx, "%", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(literalPercent) != 1 || literalPercent[0].ID != percentNote.ID {
		t.Fatalf("literal percent search results: %#v", literalPercent)
	}
	literalUnderscore, err := st.SearchNotes(ctx, "_", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(literalUnderscore) != 0 {
		t.Fatalf("literal underscore search results: %#v", literalUnderscore)
	}
}
