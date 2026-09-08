package exporter

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestMarkdownDistinctStableOutputsAndSummary(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	titles := []string{"Standup", "Standup", "a/b", "a?b", strings.Repeat("x", 81) + "a", strings.Repeat("x", 81) + "b"}
	summary := "Archived public summary"
	empty := ""
	for i, title := range titles {
		note := model.Note{ID: "note/" + string(rune('a'+i)), Type: "meeting", Title: &title, CreatedAt: now.Add(time.Duration(i) * time.Second), UpdatedAt: now, LastSeenAt: now, Source: model.SourcePublicAPI, SummaryText: &summary}
		if i%2 == 0 {
			note.SummaryMarkdown = &empty
		}
		if err := st.UpsertNote(ctx, note); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	unrelated := filepath.Join(out, "unrelated.md")
	if err := os.WriteFile(unrelated, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := Markdown(ctx, st, out, 100)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Markdown(ctx, st, out, 100)
	if err != nil || !reflect.DeepEqual(first, second) || first.Count != len(titles) {
		t.Fatalf("unstable result: %+v %+v %v", first, second, err)
	}
	seen := map[string]bool{}
	for _, path := range first.Files {
		if seen[path] || filepath.Dir(path) != out {
			t.Fatalf("duplicate or escaping path: %q", path)
		}
		seen[path] = true
		raw, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(raw), "## Summary\n\n"+summary) {
			t.Fatalf("missing summary: %s %v", raw, err)
		}
	}
	raw, err := os.ReadFile(unrelated)
	if err != nil || string(raw) != "untouched" {
		t.Fatalf("unrelated output modified: %v", err)
	}
	entries, err := os.ReadDir(out)
	if err != nil || len(entries) != len(titles)+1 {
		t.Fatalf("file count: %d %v", len(entries), err)
	}
}
