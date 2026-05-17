package exporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

type MarkdownResult struct {
	OutDir string   `json:"out_dir"`
	Count  int      `json:"count"`
	Files  []string `json:"files"`
}

func Markdown(ctx context.Context, st *store.Store, outDir string, limit int) (MarkdownResult, error) {
	if outDir == "" {
		return MarkdownResult{}, fmt.Errorf("export markdown requires --out <dir>")
	}
	notes, err := st.ListNotes(ctx, limit)
	if err != nil {
		return MarkdownResult{}, err
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return MarkdownResult{}, err
	}
	result := MarkdownResult{OutDir: outDir}
	for _, note := range notes {
		path := filepath.Join(outDir, noteFilename(note))
		if err := os.WriteFile(path, []byte(renderNote(ctx, st, note)), 0o600); err != nil {
			return MarkdownResult{}, err
		}
		result.Files = append(result.Files, path)
	}
	result.Count = len(result.Files)
	return result, nil
}

func renderNote(ctx context.Context, st *store.Store, note model.Note) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", valueOr(note.Title, note.ID))
	fmt.Fprintf(&b, "- id: `%s`\n", note.ID)
	fmt.Fprintf(&b, "- source: `%s`\n", note.Source)
	fmt.Fprintf(&b, "- created: `%s`\n", note.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(&b, "- updated: `%s`\n\n", note.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if text := valueOr(note.NotesMarkdown, ""); text != "" {
		b.WriteString(text)
		b.WriteString("\n\n")
	} else if text := valueOr(note.NotesPlain, ""); text != "" {
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	if chunks, err := st.ListTranscript(ctx, note.ID); err == nil && len(chunks) > 0 {
		b.WriteString("## Transcript\n\n")
		for _, chunk := range chunks {
			fmt.Fprintf(&b, "- `%s` %s\n", chunk.StartTimestamp.Format("15:04:05"), chunk.Text)
		}
		b.WriteString("\n")
	}
	if panels, err := st.ListPanels(ctx, note.ID); err == nil && len(panels) > 0 {
		b.WriteString("## Panels\n\n")
		for _, panel := range panels {
			fmt.Fprintf(&b, "### %s\n\n", valueOr(panel.Title, panel.ID))
			if text := valueOr(panel.ContentMarkdown, ""); text != "" {
				b.WriteString(text)
				b.WriteString("\n\n")
			} else if text := valueOr(panel.ContentPlain, ""); text != "" {
				b.WriteString(text)
				b.WriteString("\n\n")
			}
		}
	}
	return b.String()
}

func noteFilename(note model.Note) string {
	title := safeFilename(valueOr(note.Title, note.ID))
	date := note.CreatedAt.Format("2006-01-02")
	return fmt.Sprintf("%s-%s.md", date, title)
}

var filenameCleaner = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "untitled"
	}
	value = filenameCleaner.ReplaceAllString(value, "-")
	value = strings.Trim(value, ".-_")
	if len(value) > 80 {
		value = value[:80]
	}
	if value == "" {
		return "untitled"
	}
	return value
}

func valueOr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}
