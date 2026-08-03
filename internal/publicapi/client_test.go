package publicapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClientPaginatesAndIncludesTranscript(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/notes":
			_, _ = w.Write([]byte(`{"notes":[],"hasMore":false,"cursor":null}`))
		case "/v1/notes/not_12345678901234":
			_, _ = w.Write([]byte(`{"id":"not_12345678901234","object":"note","title":null,"created_at":"2026-08-03T01:00:00Z","updated_at":"2026-08-03T02:00:00Z","calendar_event":{},"attendees":[],"folder_membership":[],"summary_text":"","summary_markdown":null,"transcript":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := Client{BaseURL: srv.URL, APIKey: "test-key", RequestInterval: -1}
	if _, err := client.ListNotes(context.Background(), "next cursor", 30); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetNote(context.Background(), "not_12345678901234", true); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0] != "/v1/notes?cursor=next+cursor&page_size=30" || requests[1] != "/v1/notes/not_12345678901234?include=transcript" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestClientErrorDoesNotExposeResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"secret":"must-not-leak"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	client := Client{BaseURL: srv.URL, APIKey: "test-key", RequestInterval: -1}
	_, err := client.ListNotes(context.Background(), "", 1)
	if err == nil || strings.Contains(err.Error(), "must-not-leak") {
		t.Fatalf("unsafe error = %v", err)
	}
}

func TestClientRetriesRateLimitWithoutExposingBody(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, `{"secret":"must-not-leak"}`, http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"notes":[],"hasMore":false,"cursor":null}`))
	}))
	defer srv.Close()
	client := Client{BaseURL: srv.URL, APIKey: "test-key", RequestInterval: -1}
	if _, err := client.ListNotes(context.Background(), "", 1); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestNormalizersPreserveSummaryAndStableTranscriptIdentity(t *testing.T) {
	title := "Planning"
	markdown := "summary"
	note, err := NoteToModel(Note{
		ID:              "not_12345678901234",
		Title:           &title,
		CreatedAt:       "2026-08-03T01:00:00Z",
		UpdatedAt:       "2026-08-03T02:00:00Z",
		SummaryText:     "plain",
		SummaryMarkdown: &markdown,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if note.Source != "public-api" || note.SummaryText == nil || *note.SummaryText != "plain" || note.SummaryMarkdown == nil || *note.SummaryMarkdown != "summary" {
		t.Fatalf("note = %#v", note)
	}
	input := Transcript{
		Speaker:   Speaker{Source: "microphone", Attribution: "me"},
		Text:      "hello",
		StartTime: "2026-08-03T01:00:00Z",
		EndTime:   "2026-08-03T01:00:01Z",
	}
	first, err := TranscriptToModel(note.ID, input, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := TranscriptToModel(note.ID, input, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.Source != "microphone" || !first.IsFinal {
		t.Fatalf("chunks = %#v %#v", first, second)
	}
	corrected := input
	corrected.Text = "corrected"
	corrected.Speaker.Attribution = "Speaker 1"
	replacement, err := TranscriptToModel(note.ID, corrected, 0)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.ID != first.ID || replacement.PayloadHash == first.PayloadHash {
		t.Fatalf("corrected chunk identity changed or payload hash did not: %#v %#v", first, replacement)
	}
}

func TestLivePublicAPIContract(t *testing.T) {
	if os.Getenv("GRAINCRAWL_LIVE_PUBLIC_API") != "1" {
		t.Skip("set GRAINCRAWL_LIVE_PUBLIC_API=1 to run")
	}
	key := os.Getenv(KeyEnv)
	if key == "" {
		t.Fatal("live public API test requires GRANOLA_PUBLIC_API_KEY")
	}
	client := Client{APIKey: key}
	page, err := client.ListNotes(context.Background(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Notes) != 1 || page.Notes[0].ID == "" {
		t.Fatal("live public API returned no notes")
	}
	note, err := client.GetNote(context.Background(), page.Notes[0].ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if note.ID != page.Notes[0].ID {
		t.Fatal("live note detail did not match list identity")
	}
	if _, err := NoteToModel(note, time.Now()); err != nil {
		t.Fatal("live note detail did not normalize")
	}
	for occurrence, transcript := range note.Transcript {
		if _, err := TranscriptToModel(note.ID, transcript, occurrence); err != nil {
			t.Fatal("live transcript did not normalize")
		}
	}
}
