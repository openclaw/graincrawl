package syncer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestRunHonorsSkipTranscriptsForDesktopCache(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"cache":{"version":6,"state":{"documents":{"doc1":{"id":"doc1","created_at":"2026-05-06T01:00:00Z","updated_at":"2026-05-06T02:00:00Z","title":"Test","type":"meeting","notes_plain":"plain"}},"transcripts":{"doc1":[{"document_id":"doc1","start_timestamp":"2026-05-06T01:00:01Z","end_timestamp":"2026-05-06T01:00:02Z","text":"hello","source":"microphone","id":"c1","is_final":true}]}}}}`
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, filepath.Join(root, "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Config{
		Granola: config.GranolaConfig{
			ProfilePath:       profile,
			AllowDesktopCache: true,
		},
		Sync: config.SyncConfig{
			DefaultLimit:       100,
			IncludeTranscripts: true,
		},
	}
	result, err := Run(ctx, cfg, st, Options{
		Source:          model.SourceDesktopCache,
		SkipTranscripts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Notes != 1 || result.Transcripts != 0 {
		t.Fatalf("expected one note and no transcripts, got %#v", result)
	}
	status, err := st.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Notes != 1 || status.Transcripts != 0 {
		t.Fatalf("expected archived note without transcript, got %#v", status)
	}
}
