package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runSearch(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	query := strings.TrimSpace(strings.Join(nonFlagArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: graincrawl search <query> [--limit <n>]")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	notes, err := rt.Store.SearchNotes(ctx, query, parseLimit(args, 50))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"query": query, "notes": notes})
	}
	for _, note := range notes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", note.ID, stringValue(note.Title), note.UpdatedAt.Format("2006-01-02"))
	}
	return nil
}

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}
