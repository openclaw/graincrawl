package cli

import (
	"context"
	"io"
	"strings"
	"time"

	cktui "github.com/vincentkoc/crawlkit/tui"
	"github.com/vincentkoc/graincrawl/internal/model"
	gruntime "github.com/vincentkoc/graincrawl/internal/runtime"
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
		rows = append(rows, noteRow(note))
	}
	return rows, nil
}

func noteRow(note model.Note) cktui.Row {
	title := stringValue(note.Title)
	if title == "" {
		title = note.ID
	}
	text := firstText(note.NotesMarkdown, note.NotesPlain, note.SummaryMarkdown, note.SummaryText)
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
		Text:      text,
		Detail:    text,
		CreatedAt: fields["created_at"],
		UpdatedAt: fields["updated_at"],
		Tags:      noteTags(note),
		Fields:    fields,
	}
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
