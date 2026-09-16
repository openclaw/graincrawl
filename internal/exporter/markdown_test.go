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

func TestMarkdownReadFailurePreservesExistingExport(t *testing.T) {
	for _, table := range []string{"transcript_chunks", "document_panels"} {
		t.Run(table, func(t *testing.T) {
			ctx := context.Background()
			st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			note := model.Note{ID: "doc-1", Type: "meeting"}
			if err := st.UpsertNote(ctx, note); err != nil {
				t.Fatal(err)
			}
			out := t.TempDir()
			first, err := Markdown(ctx, st, out, 10)
			if err != nil || first.Count != 1 {
				t.Fatalf("empty child sections are valid: %+v %v", first, err)
			}
			original, err := os.ReadFile(first.Files[0])
			if err != nil {
				t.Fatal(err)
			}
			title := "Changed title inside the file"
			note.NotesPlain = &title
			if err := st.UpsertNote(ctx, note); err != nil {
				t.Fatal(err)
			}
			if _, err := st.DB().ExecContext(ctx, "DROP TABLE "+table); err != nil {
				t.Fatal(err)
			}
			if result, err := Markdown(ctx, st, out, 10); err == nil || !strings.Contains(err.Error(), table) {
				t.Fatalf("missing child data must fail export: %+v %v", result, err)
			}
			got, err := os.ReadFile(first.Files[0])
			if err != nil || string(got) != string(original) {
				t.Fatalf("failed render changed existing export: %q %v", got, err)
			}
		})
	}
}
