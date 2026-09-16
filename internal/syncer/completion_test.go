package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/cachev6"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/privateapi"
	"github.com/openclaw/graincrawl/internal/publicapi"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestSyncReportsCompletionWriteFailure(t *testing.T) {
	const document = `{"id":"doc-1","type":"meeting","created_at":"2026-09-15T10:00:00Z","updated_at":"2026-09-15T10:00:00Z"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/get-documents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[`+document+`]}`)
	})
	mux.HandleFunc("/v1/get-documents-batch", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"docs":[]}`)
	})
	mux.HandleFunc("/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"notes":[{"id":"doc-1"}],"hasMore":false}`)
	})
	mux.HandleFunc("/v1/notes/doc-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, document)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	raw := []byte(`{"cache":{"version":8,"state":{"documents":{"doc-1":` + document + `}}}}`)
	for _, source := range []model.Source{model.SourcePrivateAPI, model.SourcePublicAPI, model.SourceDesktopCache, model.SourceEncryptedJSON} {
		t.Run(string(source), func(t *testing.T) {
			ctx := context.Background()
			st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			if _, err := st.DB().ExecContext(ctx, `CREATE TRIGGER fail_sync_run BEFORE INSERT ON sync_runs BEGIN SELECT RAISE(ABORT, 'synthetic run write failure'); END`); err != nil {
				t.Fatal(err)
			}
			sync := func() (Result, error) {
				switch source {
				case model.SourcePrivateAPI:
					return syncPrivateWithMessage(ctx, privateapi.Client{BaseURL: srv.URL}, st, Options{}, false, "")
				case model.SourcePublicAPI:
					return syncPublic(ctx, &publicapi.Client{BaseURL: srv.URL, RequestInterval: -1}, st, Options{})
				case model.SourceEncryptedJSON:
					return encryptedDesktopCache(ctx, st, Options{}, raw)
				default:
					file, err := cachev6.Parse(raw)
					if err != nil {
						t.Fatal(err)
					}
					return importDesktopCache(ctx, st, Options{}, file, source, "")
				}
			}
			result, err := sync()
			if err == nil || !strings.Contains(err.Error(), "synthetic run write failure") {
				t.Fatalf("sync must report completion write failure; result=%+v err=%v", result, err)
			}
			if result.Notes != 1 || result.Source != source {
				t.Fatalf("lost partial result: %+v", result)
			}
			if _, ok, err := st.GetNote(ctx, "doc-1"); err != nil || !ok {
				t.Fatalf("completed note write must remain: exists=%v err=%v", ok, err)
			}
			assertNoOKSyncRun(t, ctx, st)
			if _, err := st.DB().ExecContext(ctx, `DROP TRIGGER fail_sync_run`); err != nil {
				t.Fatal(err)
			}
			if _, err := sync(); err != nil {
				t.Fatal(err)
			}
			runs, err := st.ListSyncRuns(ctx, 10)
			if err != nil || len(runs) != 1 || runs[0].Status != "ok" || runs[0].Notes != 1 || runs[0].Source != source {
				t.Fatalf("retry completion: runs=%+v err=%v", runs, err)
			}
		})
	}
}
