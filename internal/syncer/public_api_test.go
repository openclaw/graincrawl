package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/publicapi"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestSyncPublicArchivesSummariesAndTranscriptsAcrossPages(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("cursor") {
		case "":
			writeJSON(t, w, `{"notes":[{"id":"not_11111111111111","object":"note","title":"First","owner":{},"created_at":"2026-08-03T01:00:00Z","updated_at":"2026-08-03T02:00:00Z"}],"hasMore":true,"cursor":"next"}`)
		case "next":
			writeJSON(t, w, `{"notes":[{"id":"not_22222222222222","object":"note","title":"Second","owner":{},"created_at":"2026-08-03T03:00:00Z","updated_at":"2026-08-03T04:00:00Z"}],"hasMore":false,"cursor":null}`)
		default:
			t.Fatalf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	})
	mux.HandleFunc("/v1/notes/not_11111111111111", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("include") != "transcript" {
			t.Fatalf("include = %q", r.URL.Query().Get("include"))
		}
		writeJSON(t, w, `{"id":"not_11111111111111","object":"note","title":"First","owner":{},"created_at":"2026-08-03T01:00:00Z","updated_at":"2026-08-03T02:00:00Z","calendar_event":{"calendar_event_id":"event-1"},"attendees":[],"folder_membership":[],"summary_text":"plain summary","summary_markdown":"## Summary","transcript":[{"speaker":{"source":"microphone","attribution":"me"},"text":"archived transcript","start_time":"2026-08-03T01:00:00Z","end_time":"2026-08-03T01:00:01Z"}]}`)
	})
	mux.HandleFunc("/v1/notes/not_22222222222222", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"id":"not_22222222222222","object":"note","title":"Second","owner":{},"created_at":"2026-08-03T03:00:00Z","updated_at":"2026-08-03T04:00:00Z","calendar_event":{},"attendees":[],"folder_membership":[],"summary_text":"","summary_markdown":null,"transcript":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := &publicapi.Client{BaseURL: srv.URL, APIKey: "test-key", RequestInterval: -1}
	result, err := syncPublic(ctx, client, st, Options{Limit: 2, IncludeTranscripts: true, IncludePanels: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != model.SourcePublicAPI || result.Notes != 2 || result.Transcripts != 1 || result.Panels != 0 || !strings.Contains(result.Message, "does not expose panels") {
		t.Fatalf("result = %#v", result)
	}
	note, ok, err := st.GetNote(ctx, "not_11111111111111")
	if err != nil || !ok {
		t.Fatalf("note: ok=%v err=%v", ok, err)
	}
	if note.Source != model.SourcePublicAPI || note.SummaryMarkdown == nil || *note.SummaryMarkdown != "## Summary" || note.CalendarEventID == nil || *note.CalendarEventID != "event-1" {
		t.Fatalf("note = %#v", note)
	}
	chunks, err := st.ListTranscript(ctx, note.ID)
	if err != nil || len(chunks) != 1 || chunks[0].Text != "archived transcript" {
		t.Fatalf("chunks = %#v err=%v", chunks, err)
	}
	objects, err := st.ListSourceObjects(ctx, "", 10)
	if err != nil || len(objects) != 3 {
		t.Fatalf("source objects = %#v err=%v", objects, err)
	}
}

func TestRunPublicRequiresExplicitEnableAndEnvironmentKey(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	cfg := config.Config{Granola: config.GranolaConfig{AllowPublicAPI: false}}
	if _, err := Run(ctx, cfg, st, Options{Source: model.SourcePublicAPI}); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled error = %v", err)
	}
	cfg.Granola.AllowPublicAPI = true
	t.Setenv(publicapi.KeyEnv, "")
	if _, err := Run(ctx, cfg, st, Options{Source: model.SourcePublicAPI}); err != ErrPublicAPIKeyNotFound {
		t.Fatalf("missing key error = %v", err)
	}
}

func TestRunPublicIgnoresEncryptedLocalGranolaState(t *testing.T) {
	ctx := context.Background()
	profile := t.TempDir()
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json.enc"), []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"notes":[],"hasMore":false,"cursor":null}`)
	}))
	defer srv.Close()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	t.Setenv(publicapi.KeyEnv, "test-key")
	cfg := config.Config{
		Granola: config.GranolaConfig{
			ProfilePath:     profile,
			AllowPublicAPI:  true,
			PreferredSource: string(model.SourcePublicAPI),
		},
		API:  config.APIConfig{PublicBaseURL: srv.URL},
		Sync: config.SyncConfig{DefaultLimit: 100},
	}
	result, err := Run(ctx, cfg, st, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != model.SourcePublicAPI || result.Notes != 0 {
		t.Fatalf("result = %#v", result)
	}
}
