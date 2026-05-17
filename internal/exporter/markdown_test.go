package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestMarkdownExportsNoteTranscriptAndPanels(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	title := "Demo Call"
	body := "## Decisions\n\nShip it."
	if err := st.UpsertNote(ctx, model.Note{
		ID:            "doc-1",
		Title:         &title,
		Type:          "meeting",
		CreatedAt:     now,
		UpdatedAt:     now,
		NotesMarkdown: &body,
		Source:        model.SourcePrivateAPI,
		LastSeenAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
		ID:             "chunk-1",
		DocumentID:     "doc-1",
		StartTimestamp: now,
		EndTimestamp:   now.Add(time.Second),
		Source:         "mic",
		Text:           "hello from transcript",
	}); err != nil {
		t.Fatal(err)
	}
	panelText := "panel text"
	if err := st.UpsertPanel(ctx, model.Panel{
		ID:              "panel-1",
		DocumentID:      "doc-1",
		Title:           &title,
		ContentMarkdown: &panelText,
		CreatedAt:       now,
		Source:          model.SourcePrivateAPI,
	}); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	result, err := Markdown(ctx, st, outDir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Fatalf("expected 1 file, got %d", result.Count)
	}
	got, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{"# Demo Call", "Ship it.", "hello from transcript", "panel text"} {
		if !strings.Contains(text, want) {
			t.Fatalf("export missing %q:\n%s", want, text)
		}
	}
}
