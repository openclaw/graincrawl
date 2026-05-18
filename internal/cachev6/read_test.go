package cachev6

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadSummarizeAndNormalize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache-v6.json")
	raw := `{"cache":{"version":6,"state":{"featureFlags":{"encrypted_cache_storage":true},"documents":{"doc1":{"id":"doc1","created_at":"2026-05-06T01:00:00Z","updated_at":"2026-05-06T02:00:00Z","title":"Test","type":"meeting","notes_plain":"plain"}},"transcripts":{"doc1":[{"document_id":"doc1","start_timestamp":"2026-05-06T01:00:01Z","end_timestamp":"2026-05-06T01:00:02Z","text":"hello","source":"microphone","id":"c1","is_final":true}]}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	summary := Summarize(file)
	if summary.DocumentCount != 1 || summary.TranscriptDocs != 1 || !summary.EncryptedCache {
		t.Fatalf("bad summary: %#v", summary)
	}
	note, err := NoteFromDocument(file.Cache.State.Documents["doc1"], time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if note.ID != "doc1" || note.NotesPlain == nil {
		t.Fatalf("bad note: %#v", note)
	}
}

func TestReadAcceptsEmptyVersion8Cache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache-v6.json")
	raw := `{"cache":{"version":8,"state":{"transcripts":{}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	summary := Summarize(file)
	if summary.Version != 8 || summary.DocumentCount != 0 || summary.TranscriptDocs != 0 || summary.MeetingMetaCount != 0 {
		t.Fatalf("bad empty v8 summary: %#v", summary)
	}
	if file.Cache.State.Documents == nil || file.Cache.State.Transcripts == nil || file.Cache.State.MeetingsMetadata == nil {
		t.Fatalf("expected empty maps to be initialized: %#v", file.Cache.State)
	}
}
