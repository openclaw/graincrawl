package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/model"
)

func TestPrintNoteSummary(t *testing.T) {
	markdown, plain, body := "**summary**", "plain summary", "notes body"
	for _, note := range []model.Note{
		{SummaryText: &plain},
		{NotesMarkdown: &body, SummaryMarkdown: &markdown, SummaryText: &plain},
		{NotesPlain: &body},
	} {
		var out bytes.Buffer
		printNote(&out, note)
		text := out.String()
		if note.NotesMarkdown != nil || note.NotesPlain != nil {
			if !strings.Contains(text, body) {
				t.Fatal("lost notes")
			}
		}
		if note.SummaryMarkdown != nil {
			if !strings.Contains(text, markdown) || strings.Contains(text, plain) {
				t.Fatal("summary preference lost")
			}
		} else if note.SummaryText != nil && !strings.Contains(text, plain) {
			t.Fatal("plain summary missing")
		} else if note.SummaryText == nil && strings.Contains(text, "Summary") {
			t.Fatal("empty summary section")
		}
	}
}
