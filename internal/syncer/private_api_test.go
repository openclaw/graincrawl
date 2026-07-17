package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

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
	if _, err := syncPrivateWithMessage(ctx, client, st, Options{Source: model.SourcePrivateAPI, Limit: 1}, false, ""); err != nil {
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

func TestSyncPrivateConsumesExplicitDeleteFeedAndTombstonesChildren(t *testing.T) {
	ctx := context.Background()
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		if deleted {
			writeJSON(t, w, `{"docs":[],"deleted":["doc-1"],"shared":[]}`)
			return
		}
		writeJSON(t, w, `{"docs":[{"id":"doc-1","title":"Planning","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:01:00Z"}],"deleted":[],"shared":[]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		if deleted {
			writeJSON(t, w, `{"docs":[]}`)
			return
		}
		writeJSON(t, w, `{"docs":[{"id":"doc-1","title":"Planning","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:02:00Z"}]}`)
	})
	mux.HandleFunc("/v1/get-document-transcript", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `[{"id":"chunk-1","document_id":"doc-1","start_timestamp":"2026-05-06T10:00:00Z","end_timestamp":"2026-05-06T10:00:01Z","source":"mic","is_final":true,"text":"retained transcript"}]`)
	})
	mux.HandleFunc("/v1/get-document-panels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `[{"id":"panel-1","document_id":"doc-1","created_at":"2026-05-06T10:00:00Z","title":"Summary","content":{"text":"retained panel"}}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := privateapi.Client{BaseURL: srv.URL, AccessToken: "token"}
	opts := Options{Source: model.SourcePrivateAPI, IncludeTranscripts: true, IncludePanels: true}
	if _, err := syncPrivateWithMessage(ctx, client, st, opts, false, ""); err != nil {
		t.Fatal(err)
	}
	deleted = true
	result, err := syncPrivateWithMessage(ctx, client, st, opts, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", result.Deleted)
	}

	note, ok, err := st.GetNote(ctx, "doc-1")
	if err != nil || !ok {
		t.Fatalf("get tombstoned note: ok=%v err=%v", ok, err)
	}
	assertDeletion(t, "note", note.DeletedAt, note.DeletionSource, note.DeletionReason)
	chunks, err := st.ListTranscript(ctx, "doc-1")
	if err != nil || len(chunks) != 1 {
		t.Fatalf("transcript chunks = %#v, err=%v", chunks, err)
	}
	assertDeletion(t, "transcript", chunks[0].DeletedAt, chunks[0].DeletionSource, chunks[0].DeletionReason)
	panels, err := st.ListPanels(ctx, "doc-1")
	if err != nil || len(panels) != 1 {
		t.Fatalf("panels = %#v, err=%v", panels, err)
	}
	assertDeletion(t, "panel", panels[0].DeletedAt, panels[0].DeletionSource, panels[0].DeletionReason)
	objects, err := st.ListSourceObjects(ctx, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 3 {
		t.Fatalf("source objects = %d, want 3: %#v", len(objects), objects)
	}
	for _, object := range objects {
		assertDeletion(t, fmt.Sprintf("source object %s", object.Kind), object.DeletedAt, object.DeletionSource, object.DeletionReason)
	}
}

func TestSyncPrivateDoesNotDeleteNoteMerelyBecauseItWasNotSeen(t *testing.T) {
	ctx := context.Background()
	missing := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		if missing {
			writeJSON(t, w, `{"docs":[],"deleted":[],"shared":[]}`)
			return
		}
		writeJSON(t, w, `{"docs":[{"id":"doc-1","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:01:00Z"}],"deleted":[],"shared":[]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := privateapi.Client{BaseURL: srv.URL, AccessToken: "token"}
	if _, err := syncPrivateWithMessage(ctx, client, st, Options{}, false, ""); err != nil {
		t.Fatal(err)
	}
	missing = true
	if _, err := syncPrivateWithMessage(ctx, client, st, Options{}, false, ""); err != nil {
		t.Fatal(err)
	}
	note, ok, err := st.GetNote(ctx, "doc-1")
	if err != nil || !ok {
		t.Fatalf("get retained note: ok=%v err=%v", ok, err)
	}
	if note.DeletedAt != nil || note.DeletionSource != "" || note.DeletionReason != "" {
		t.Fatalf("not-seen note was tombstoned: %#v", note)
	}
}

func TestSyncPrivateRetainsUnknownExplicitDeleteAsStubTombstone(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[],"deleted":["unknown-doc"],"shared":[]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := privateapi.Client{BaseURL: srv.URL, AccessToken: "token"}
	if _, err := syncPrivateWithMessage(ctx, client, st, Options{}, false, ""); err != nil {
		t.Fatal(err)
	}
	note, ok, err := st.GetNote(ctx, "unknown-doc")
	if err != nil || !ok {
		t.Fatalf("get stub tombstone: ok=%v err=%v", ok, err)
	}
	if note.Type != "unknown" {
		t.Fatalf("stub type = %q, want unknown", note.Type)
	}
	assertDeletion(t, "stub note", note.DeletedAt, note.DeletionSource, note.DeletionReason)
}

func TestSyncPrivateRetainsPanelDeletedAtOnPanelAndSourceObject(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[{"id":"doc-1","type":"meeting","created_at":"2026-05-06T10:00:00Z","updated_at":"2026-05-06T10:01:00Z"}],"deleted":[],"shared":[]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[]}`)
	})
	mux.HandleFunc("/v1/get-document-panels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `[{"id":"panel-1","document_id":"doc-1","created_at":"2026-05-06T10:00:00Z","deleted_at":"2026-05-07T11:00:00Z"}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "graincrawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := privateapi.Client{BaseURL: srv.URL, AccessToken: "token"}
	if _, err := syncPrivateWithMessage(ctx, client, st, Options{IncludePanels: true}, false, ""); err != nil {
		t.Fatal(err)
	}
	panels, err := st.ListPanels(ctx, "doc-1")
	if err != nil || len(panels) != 1 {
		t.Fatalf("panels = %#v err=%v", panels, err)
	}
	if panels[0].DeletedAt == nil || panels[0].DeletionReason != store.DeletionReasonSourceField {
		t.Fatalf("panel deletion = %#v", panels[0])
	}
	objects, err := st.ListSourceObjects(ctx, "panel", 10)
	if err != nil || len(objects) != 1 {
		t.Fatalf("panel source objects = %#v err=%v", objects, err)
	}
	if objects[0].DeletedAt == nil || objects[0].DeletionReason != store.DeletionReasonSourceField {
		t.Fatalf("panel source deletion = %#v", objects[0])
	}
}

func assertDeletion(t *testing.T, label string, deletedAt *time.Time, source, reason string) {
	t.Helper()
	if deletedAt == nil || source != string(model.SourcePrivateAPI) || reason != store.DeletionReasonExplicitFeed {
		t.Fatalf("%s deletion = (%v, %q, %q)", label, deletedAt, source, reason)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}
