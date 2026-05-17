package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/graincrawl/internal/hashutil"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func retainSourceObject(ctx context.Context, st *store.Store, source model.Source, kind, sourceID, documentID string, payload any, observed time.Time) error {
	if sourceID == "" {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s %s: %w", kind, sourceID, err)
	}
	return st.UpsertSourceObject(ctx, store.SourceObject{
		Source:      source,
		Kind:        kind,
		SourceID:    sourceID,
		DocumentID:  documentID,
		PayloadJSON: string(data),
		PayloadHash: hashutil.JSON(payload),
		ObservedAt:  observed,
	})
}

func retainPeople(ctx context.Context, st *store.Store, source model.Source, documentID string, raw json.RawMessage, observed time.Time) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var people []map[string]any
	if err := json.Unmarshal(raw, &people); err != nil {
		return retainSourceObject(ctx, st, source, "people", documentID+":people", documentID, raw, observed)
	}
	for i, person := range people {
		id := stringField(person, "id")
		if id == "" {
			id = stringField(person, "user_id")
		}
		if id == "" {
			id = fmt.Sprintf("%s:person:%d", documentID, i)
		}
		if err := retainSourceObject(ctx, st, source, "person", id, documentID, person, observed); err != nil {
			return err
		}
	}
	return nil
}

func stringField(value map[string]any, key string) string {
	raw, ok := value[key]
	if !ok {
		return ""
	}
	text, _ := raw.(string)
	return text
}
