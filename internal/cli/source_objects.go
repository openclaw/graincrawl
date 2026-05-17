package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runRuns(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	runs, err := rt.Store.ListSyncRuns(ctx, parseLimit(args, 20))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"runs": runs})
	}
	for _, run := range runs {
		fmt.Fprintf(w, "%d\t%s\t%s\tnotes=%d transcripts=%d panels=%d\n",
			run.ID, run.Source, run.CompletedAt.Format("2006-01-02 15:04:05"), run.Notes, run.Transcripts, run.Panels)
	}
	return nil
}

func (a App) runSourceObjectList(ctx context.Context, w io.Writer, flags GlobalFlags, args []string, kind string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	objects, err := rt.Store.ListSourceObjects(ctx, kind, parseLimit(args, 100))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"kind": kind, "objects": objects})
	}
	for _, obj := range objects {
		fmt.Fprintf(w, "%s\t%s\t%s\n", obj.SourceID, obj.Source, obj.ObservedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}
