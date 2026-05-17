package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/privateapi"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestSyncPrivateHydratesDocumentBodyBeforeUpsert(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[{"id":"doc-1","title":"Planning","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:01:00Z"}],"deleted":[],"shared":[]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[{"id":"doc-1","title":"Planning","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:02:00Z","notes_markdown":"hydrated note body"}]}`)
	})
	mux.HandleFunc("/v1/get-document-transcript", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `[]`)
	})
	mux.HandleFunc("/v1/get-document-panels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := privateapi.Client{BaseURL: srv.URL, AccessToken: "token"}
	if _, err := syncPrivate(ctx, client, st, Options{Source: model.SourcePrivateAPI, Limit: 1}, false); err != nil {
		t.Fatal(err)
	}
	note, ok, err := st.GetNote(ctx, "doc-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || note.NotesMarkdown == nil || *note.NotesMarkdown != "hydrated note body" {
		t.Fatalf("expected hydrated note body, got %#v", note)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}
