package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runNotes(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	notes, err := rt.Store.ListNotes(ctx, parseLimit(args, 100))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"notes": notes})
	}
	for _, note := range notes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", note.ID, stringValue(note.Title), note.CreatedAt.Format("2006-01-02"))
	}
	return nil
}

func (a App) runNote(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	if len(args) != 2 || args[0] != "get" {
		return fmt.Errorf("usage: graincrawl note get <id>")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	note, ok, err := rt.Store.GetNote(ctx, args[1])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("note %q not found", args[1])
	}
	if flags.JSON {
		return output.WriteEnvelope(w, note)
	}
	printNote(w, note)
	return nil
}

func (a App) runTranscripts(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	if len(args) != 2 || args[0] != "get" {
		return fmt.Errorf("usage: graincrawl transcripts get <id>")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	chunks, err := rt.Store.ListTranscript(ctx, args[1])
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"document_id": args[1], "chunks": chunks})
	}
	for _, chunk := range chunks {
		fmt.Fprintf(w, "[%s] %s\n", chunk.StartTimestamp.Format("15:04:05"), chunk.Text)
	}
	return nil
}

func (a App) runPanels(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	if len(args) != 2 || args[0] != "get" {
		return fmt.Errorf("usage: graincrawl panels get <id>")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	panels, err := rt.Store.ListPanels(ctx, args[1])
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"document_id": args[1], "panels": panels})
	}
	for _, panel := range panels {
		fmt.Fprintf(w, "%s\t%s\n", panel.ID, stringValue(panel.Title))
		if text := stringValue(panel.ContentMarkdown); text != "" {
			fmt.Fprintln(w, text)
		} else if text := stringValue(panel.ContentPlain); text != "" {
			fmt.Fprintln(w, text)
		}
	}
	return nil
}

func printNote(w io.Writer, note model.Note) {
	output.PrintKV(w, "id", note.ID)
	output.PrintKV(w, "title", stringValue(note.Title))
	output.PrintKV(w, "created", note.CreatedAt.Format("2006-01-02 15:04:05"))
	if text := stringValue(note.NotesMarkdown); text != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, text)
		return
	}
	if text := stringValue(note.NotesPlain); text != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, text)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
