package syncer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRunAcceptsEmptyVersion8DesktopCache(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"cache":{"version":8,"state":{"transcripts":{}}}}`
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
	result, err := Run(ctx, cfg, st, Options{Source: model.SourceDesktopCache})
	if err != nil {
		t.Fatal(err)
	}
	if result.Notes != 0 || result.Transcripts != 0 || result.Source != model.SourceDesktopCache {
		t.Fatalf("expected empty desktop-cache sync, got %#v", result)
	}
	status, err := st.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.SyncRuns != 1 || status.Notes != 0 || status.Transcripts != 0 {
		t.Fatalf("expected empty sync run to be recorded, got %#v", status)
	}
}

func TestRunExplainsEncryptedOnlyDesktopCache(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"cache":{"version":8,"state":{"transcripts":{}}}}`
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"supabase.json.enc", "storage.dek"} {
		if err := os.WriteFile(filepath.Join(profile, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
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
		Sync: config.SyncConfig{DefaultLimit: 100},
	}
	result, err := Run(ctx, cfg, st, Options{Source: model.SourceDesktopCache})
	if err != nil {
		t.Fatal(err)
	}
	if result.Notes != 0 || !strings.Contains(result.Message, "encrypted-only") {
		t.Fatalf("expected encrypted-only diagnostic on empty sync, got %#v", result)
	}
	runs, err := st.ListSyncRuns(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Message != result.Message {
		t.Fatalf("expected diagnostic recorded with sync run, got %#v", runs)
	}
}

func TestRunDefaultSyncFallsBackWithEncryptedOnlyDiagnostic(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"cache":{"version":8,"state":{"documents":{"doc1":{"id":"doc1","created_at":"2026-05-06T01:00:00Z","updated_at":"2026-05-06T02:00:00Z","title":"Cached","type":"meeting","notes_plain":"cached plain"}},"transcripts":{}}}}`
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	supabasePath := filepath.Join(profile, "supabase.json")
	supabaseEncPath := filepath.Join(profile, "supabase.json.enc")
	supabaseRaw := `{"workos_tokens":"{\"access_token\":\"stale\",\"obtained_at\":4102444800000,\"expires_in\":3600}"}`
	if err := os.WriteFile(supabasePath, []byte(supabaseRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"supabase.json.enc", "storage.dek"} {
		if err := os.WriteFile(filepath.Join(profile, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	makeNewer(t, supabasePath, supabaseEncPath)
	st, err := store.Open(ctx, filepath.Join(root, "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Config{
		Granola: config.GranolaConfig{
			ProfilePath:       profile,
			PreferredSource:   "private-api",
			AllowPrivateAPI:   true,
			AllowDesktopCache: true,
		},
		Sync: config.SyncConfig{DefaultLimit: 100},
	}
	result, err := Run(ctx, cfg, st, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != model.SourceDesktopCache || result.Notes != 1 || !strings.Contains(result.Message, "encrypted-only") {
		t.Fatalf("expected default sync fallback with encrypted-only diagnostic, got %#v", result)
	}
}

func TestRunRejectsNewerEncryptedDesktopCache(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(profile, "cache-v6.json")
	cacheEncPath := filepath.Join(profile, "cache-v6.json.enc")
	raw := `{"cache":{"version":8,"state":{"documents":{"doc1":{"id":"doc1","created_at":"2026-05-06T01:00:00Z","updated_at":"2026-05-06T02:00:00Z","title":"Cached","type":"meeting","notes_plain":"cached plain"}},"transcripts":{}}}}`
	if err := os.WriteFile(cachePath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheEncPath, []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}
	makeNewer(t, cachePath, cacheEncPath)
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
		Sync: config.SyncConfig{DefaultLimit: 100},
	}
	result, err := Run(ctx, cfg, st, Options{Source: model.SourceDesktopCache})
	if err == nil {
		t.Fatalf("expected stale plaintext cache rejection, got %#v", result)
	}
	if !strings.Contains(err.Error(), "cache-v6.json") || !strings.Contains(err.Error(), "encrypted-only") {
		t.Fatalf("expected encrypted-cache diagnostic error, got %v", err)
	}
}

func TestRunFallsBackToDesktopCacheForImplicitExpiredPrivateAPI(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheRaw := `{"cache":{"version":8,"state":{"transcripts":{}}}}`
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json"), []byte(cacheRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	supabaseRaw := `{"workos_tokens":"{\"access_token\":\"expired\",\"obtained_at\":0,\"expires_in\":1}"}`
	if err := os.WriteFile(filepath.Join(profile, "supabase.json"), []byte(supabaseRaw), 0o600); err != nil {
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
			PreferredSource:   "private-api",
			AllowPrivateAPI:   true,
			AllowDesktopCache: true,
		},
		Sync: config.SyncConfig{DefaultLimit: 100},
	}
	result, err := Run(ctx, cfg, st, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != model.SourceDesktopCache || result.Notes != 0 || result.Transcripts != 0 {
		t.Fatalf("expected implicit expired private-api to fall back to empty desktop-cache, got %#v", result)
	}
	if _, err := Run(ctx, cfg, st, Options{Source: model.SourcePrivateAPI}); !errors.Is(err, ErrPrivateAPITokenExpired) {
		t.Fatalf("explicit private-api should still fail with token expiry, got %v", err)
	}
}

func makeNewer(t *testing.T, olderPath, newerPath string) {
	t.Helper()
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(olderPath, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newerPath, newer, newer); err != nil {
		t.Fatal(err)
	}
}
