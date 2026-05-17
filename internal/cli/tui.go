package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	cktui "github.com/openclaw/crawlkit/tui"
	"github.com/openclaw/graincrawl/internal/model"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runTUI(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	limit := parseLimit(args, 200)
	rows, err := noteRows(ctx, rt, limit)
	if err != nil {
		return err
	}
	return cktui.Browse(ctx, cktui.BrowseOptions{
		AppName:        "graincrawl",
		Title:          "Granola archive",
		EmptyMessage:   "graincrawl has no archived notes yet",
		Rows:           rows,
		Refresh:        func(ctx context.Context) ([]cktui.Row, error) { return noteRows(ctx, rt, limit) },
		RefreshEvery:   15 * time.Second,
		JSON:           flags.JSON,
		Layout:         cktui.LayoutDocument,
		SourceKind:     "notes",
		SourceLocation: rt.Store.Path(),
		Stdout:         w,
	})
}

func noteRows(ctx context.Context, rt gruntime.Runtime, limit int) ([]cktui.Row, error) {
	notes, err := rt.Store.ListNotes(ctx, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]cktui.Row, 0, len(notes))
	for _, note := range notes {
		row, err := noteRow(ctx, rt, note)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func noteRow(ctx context.Context, rt gruntime.Runtime, note model.Note) (cktui.Row, error) {
	title := stringValue(note.Title)
	if title == "" {
		title = note.ID
	}
	detail, err := noteDetail(ctx, rt, note)
	if err != nil {
		return cktui.Row{}, err
	}
	fields := map[string]string{
		"type":       note.Type,
		"source":     string(note.Source),
		"created_at": note.CreatedAt.Format(time.RFC3339),
		"updated_at": note.UpdatedAt.Format(time.RFC3339),
	}
	if note.Status != nil && *note.Status != "" {
		fields["status"] = *note.Status
	}
	if note.WorkspaceID != nil && *note.WorkspaceID != "" {
		fields["workspace_id"] = *note.WorkspaceID
	}
	return cktui.Row{
		Source:    cktui.SourceLocal,
		Kind:      "note",
		ID:        note.ID,
		Scope:     string(note.Source),
		Title:     title,
		Text:      detail,
		Detail:    detail,
		CreatedAt: fields["created_at"],
		UpdatedAt: fields["updated_at"],
		Tags:      noteTags(note),
		Fields:    fields,
	}, nil
}

func noteDetail(ctx context.Context, rt gruntime.Runtime, note model.Note) (string, error) {
	var b strings.Builder
	hasNoteText := writeSection(&b, "Notes", firstText(note.NotesMarkdown, note.NotesPlain))
	writeSection(&b, "Summary", firstText(note.SummaryMarkdown, note.SummaryText))
	panels, err := rt.Store.ListPanels(ctx, note.ID)
	if err != nil {
		return "", err
	}
	if len(panels) > 0 {
		if hasNoteText {
			writeHeading(&b, "Panels")
		} else {
			writeHeading(&b, "Notes / Panels")
		}
		for _, panel := range panels {
			fmt.Fprintf(&b, "### %s\n\n", firstNonEmpty(stringValue(panel.Title), panel.ID))
			writeBlock(&b, firstText(panel.ContentMarkdown, panel.ContentPlain))
		}
	}
	chunks, err := rt.Store.ListTranscript(ctx, note.ID)
	if err != nil {
		return "", err
	}
	if len(chunks) > 0 {
		writeHeading(&b, "Conversation")
		for _, chunk := range chunks {
			fmt.Fprintf(&b, "- `%s` %s\n", chunk.StartTimestamp.Format("15:04:05"), strings.TrimSpace(chunk.Text))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func noteTags(note model.Note) []string {
	tags := []string{string(note.Source)}
	if note.Type != "" {
		tags = append(tags, note.Type)
	}
	if note.Status != nil && *note.Status != "" {
		tags = append(tags, *note.Status)
	}
	return tags
}

func firstText(values ...*string) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text := strings.TrimSpace(*value); text != "" {
			return text
		}
	}
	return ""
}

func writeSection(b *strings.Builder, title, text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	writeHeading(b, title)
	writeBlock(b, text)
	return true
}

func writeHeading(b *strings.Builder, title string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "## %s\n\n", title)
}

func writeBlock(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	b.WriteString(text)
	b.WriteString("\n\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
