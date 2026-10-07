package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
)

func TestSearchNotesUsesUnicodeLowercase(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC().Round(0)
	notes := []struct {
		id, title string
	}{
		{"emile", "Émile sync"},
		{"oil", "Ölgeschäft review"},
		{"plain", "Plain Title"},
	}
	for _, item := range notes {
		title := item.title
		note := model.Note{
			ID: item.id, Title: &title, Type: "meeting",
			CreatedAt: now, UpdatedAt: now, Source: model.SourcePrivateAPI, LastSeenAt: now,
		}
		if err := st.UpsertNote(ctx, note); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		query string
		want  string
	}{
		{"Émile", "emile"},
		{"émile", "emile"},
		{"ÉMILE", "emile"},
		{"Ölgeschäft", "oil"},
		{"ölgeschäft", "oil"},
		{"Plain", "plain"},
		{"plain", "plain"},
	}
	for _, tc := range cases {
		results, err := st.SearchNotes(ctx, tc.query, 10)
		if err != nil {
			t.Fatalf("search %q: %v", tc.query, err)
		}
		if len(results) != 1 || results[0].ID != tc.want {
			t.Fatalf("search %q = %#v", tc.query, results)
		}
	}
}

func TestSearchNotesUnicodeAcrossFields(t *testing.T) {
	text := `ÉLAN_50%\PATH`
	cases := []struct {
		name       string
		note       model.Note
		transcript string
		panel      model.Panel
	}{
		{name: "title", note: model.Note{Title: &text}},
		{name: "notes_plain", note: model.Note{NotesPlain: &text}},
		{name: "notes_markdown", note: model.Note{NotesMarkdown: &text}},
		{name: "summary_text", note: model.Note{SummaryText: &text}},
		{name: "summary_markdown", note: model.Note{SummaryMarkdown: &text}},
		{name: "transcript", transcript: text},
		{name: "panel_plain", panel: model.Panel{ContentPlain: &text}},
		{name: "panel_markdown", panel: model.Panel{ContentMarkdown: &text}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			st, err := Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			now := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
			note := tc.note
			note.ID, note.Type, note.Source = "match", "meeting", model.SourcePrivateAPI
			note.CreatedAt, note.UpdatedAt, note.LastSeenAt = now, now, now
			if err := st.UpsertNote(ctx, note); err != nil {
				t.Fatal(err)
			}
			if tc.transcript != "" {
				if err := st.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
					ID: "chunk", DocumentID: note.ID, Text: tc.transcript,
					StartTimestamp: now, EndTimestamp: now, Source: "mic",
				}); err != nil {
					t.Fatal(err)
				}
			}
			if tc.panel.ContentPlain != nil || tc.panel.ContentMarkdown != nil {
				panel := tc.panel
				panel.ID, panel.DocumentID = "panel", note.ID
				panel.CreatedAt, panel.Source = now, model.SourcePrivateAPI
				if err := st.UpsertPanel(ctx, panel); err != nil {
					t.Fatal(err)
				}
			}
			for _, query := range []string{`élan_50%\path`, `ÉLAN_50%\PATH`} {
				results, err := st.SearchNotes(ctx, query, 10)
				if err != nil || len(results) != 1 || results[0].ID != note.ID {
					t.Fatalf("search %q = %#v, %v", query, results, err)
				}
			}
			for _, query := range []string{"missing", "é%", "é_", `é\`} {
				results, err := st.SearchNotes(ctx, query, 10)
				if err != nil || len(results) != 0 {
					t.Fatalf("literal search %q = %#v, %v", query, results, err)
				}
			}
		})
	}
}
