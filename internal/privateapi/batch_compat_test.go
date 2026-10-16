package privateapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDocumentsBatchChunks(t *testing.T) {
	for _, n := range []int{0, 1, 50, 51, 98, 101} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var b BatchRequest
				if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
					t.Errorf("decode request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				calls++
				if len(b.DocumentIDs) > 50 {
					t.Errorf("batch contains %d IDs, want at most 50", len(b.DocumentIDs))
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				docs := []Document{}
				for _, id := range b.DocumentIDs {
					docs = append(docs, Document{ID: id})
				}
				if err := json.NewEncoder(w).Encode(BatchResponse{Docs: docs}); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer srv.Close()
			ids := []string{}
			for i := 0; i < n; i++ {
				ids = append(ids, fmt.Sprint(i))
			}
			c := Client{BaseURL: srv.URL}
			got, err := c.GetDocumentsBatch(context.Background(), ids)
			if err != nil || len(got.Docs) != n || calls != (n+49)/50 {
				t.Fatalf("calls=%d len=%d err=%v", calls, len(got.Docs), err)
			}
			for i, d := range got.Docs {
				if d.ID != ids[i] {
					t.Fatalf("document %d = %q, want %q", i, d.ID, ids[i])
				}
			}
		})
	}
}
