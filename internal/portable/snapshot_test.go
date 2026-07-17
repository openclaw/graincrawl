package portable

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestImportMergesByDefaultAndReplaceIsExplicit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	source := openTestStore(t, ctx, "source.db")
	defer source.Close()
	upsertTestNote(t, ctx, source, "snapshot-only", "Snapshot note", now)
	upsertTestNote(t, ctx, source, "shared-id", "Snapshot version", now)
	deletedAt := now.Add(time.Hour)
	deletedTitle := "Legacy deleted note"
	if err := source.UpsertNote(ctx, model.Note{
		ID: "legacy-deleted", Title: &deletedTitle, Type: "meeting", CreatedAt: now,
		UpdatedAt: now, DeletedAt: &deletedAt, Source: model.SourcePrivateAPI, LastSeenAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := source.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
		ID: "legacy-chunk", DocumentID: "legacy-deleted", StartTimestamp: now,
		EndTimestamp: now.Add(time.Second), Source: "mic", Text: "retained",
	}); err != nil {
		t.Fatal(err)
	}
	if err := source.UpsertSourceObject(ctx, store.SourceObject{
		Source: model.SourcePrivateAPI, Kind: "document", SourceID: "legacy-deleted",
		DocumentID: "legacy-deleted", PayloadJSON: `{}`, PayloadHash: "hash", ObservedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	snapshotDir := filepath.Join(t.TempDir(), "snapshot")
	if _, err := Export(ctx, source, Options{RootDir: snapshotDir}); err != nil {
		t.Fatal(err)
	}

	destination := openTestStore(t, ctx, "destination.db")
	defer destination.Close()
	upsertTestNote(t, ctx, destination, "destination-only", "Local note", now)
	upsertTestNote(t, ctx, destination, "shared-id", "Local version", now)
	if err := destination.TombstoneDocument(ctx, "shared-id", store.Deletion{
		At: now.Add(time.Hour), Source: model.SourcePrivateAPI, Reason: store.DeletionReasonExplicitFeed,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := Import(ctx, destination, Options{RootDir: snapshotDir}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"destination-only", "snapshot-only", "shared-id"} {
		if _, ok, err := destination.GetNote(ctx, id); err != nil || !ok {
			t.Fatalf("merged note %s: ok=%v err=%v", id, ok, err)
		}
	}
	shared, _, err := destination.GetNote(ctx, "shared-id")
	if err != nil {
		t.Fatal(err)
	}
	if shared.DeletedAt == nil || shared.DeletionReason != store.DeletionReasonExplicitFeed {
		t.Fatalf("merge cleared tombstone: %#v", shared)
	}
	if shared.Title == nil || *shared.Title != "Local version" {
		t.Fatalf("merge overwrote destination payload: %#v", shared)
	}
	assertImportedLegacyTombstones(t, ctx, destination)

	if _, err := Import(ctx, destination, Options{RootDir: snapshotDir, Replace: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := destination.GetNote(ctx, "destination-only"); err != nil || ok {
		t.Fatalf("replace retained destination-only note: ok=%v err=%v", ok, err)
	}
	if _, ok, err := destination.GetNote(ctx, "snapshot-only"); err != nil || !ok {
		t.Fatalf("replace lost snapshot note: ok=%v err=%v", ok, err)
	}
	assertImportedLegacyTombstones(t, ctx, destination)
}

func assertImportedLegacyTombstones(t *testing.T, ctx context.Context, st *store.Store) {
	t.Helper()
	note, ok, err := st.GetNote(ctx, "legacy-deleted")
	if err != nil || !ok || note.DeletedAt == nil || note.DeletionSource != string(model.SourcePrivateAPI) || note.DeletionReason != store.DeletionReasonLegacyField {
		t.Fatalf("legacy imported note: ok=%v err=%v note=%#v", ok, err, note)
	}
	chunks, err := st.ListTranscript(ctx, "legacy-deleted")
	if err != nil || len(chunks) != 1 || chunks[0].DeletedAt == nil || chunks[0].DeletionReason != store.DeletionReasonLegacyField {
		t.Fatalf("legacy imported transcript: chunks=%#v err=%v", chunks, err)
	}
	objects, err := st.ListSourceObjects(ctx, "document", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objects {
		if object.SourceID == "legacy-deleted" && object.DeletedAt != nil && object.DeletionReason == store.DeletionReasonLegacyField {
			return
		}
	}
	t.Fatalf("legacy imported source object not tombstoned: %#v", objects)
}

func openTestStore(t *testing.T, ctx context.Context, name string) *store.Store {
	t.Helper()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func upsertTestNote(t *testing.T, ctx context.Context, st *store.Store, id, title string, now time.Time) {
	t.Helper()
	if err := st.UpsertNote(ctx, model.Note{
		ID: id, Title: &title, Type: "meeting", CreatedAt: now, UpdatedAt: now,
		Source: model.SourcePrivateAPI, LastSeenAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}
