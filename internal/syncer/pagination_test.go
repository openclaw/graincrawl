package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openclaw/graincrawl/internal/publicapi"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestSyncPublicRejectsPaginationCycles(t *testing.T) {
	for _, cursors := range [][]string{{"a", "a"}, {"a", "b", "a"}} {
		t.Run(strings.Join(cursors, "-"), func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				i := int(requests.Add(1)) - 1
				if i >= len(cursors) {
					http.Error(w, "pagination did not stop", http.StatusInternalServerError)
					return
				}
				t.Logf("GET %s -> cursor=%q, hasMore=true, notes=[]", r.URL.RequestURI(), cursors[i])
				writeJSON(t, w, fmt.Sprintf(`{"notes":[],"hasMore":true,"cursor":%q}`, cursors[i]))
			}))
			defer srv.Close()
			ctx := context.Background()
			st, err := store.Open(ctx, filepath.Join(t.TempDir(), "archive.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			client := &publicapi.Client{BaseURL: srv.URL, RequestInterval: -1}
			_, err = syncPublic(ctx, client, st, Options{})
			if err == nil || !strings.Contains(err.Error(), "invalid pagination cursor") {
				t.Fatalf("cycle error = %v", err)
			}
			if got := int(requests.Load()); got != len(cursors) {
				t.Fatalf("requested %d pages, want %d", got, len(cursors))
			}
			assertNoOKSyncRun(t, ctx, st)
			t.Logf("requests=%d, sync_error=%q; no successful sync run recorded", requests.Load(), err)
		})
	}
}
