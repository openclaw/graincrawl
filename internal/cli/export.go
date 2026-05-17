package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/openclaw/graincrawl/internal/exporter"
	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runExport(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	if len(args) == 0 || args[0] != "markdown" {
		return fmt.Errorf("usage: graincrawl export markdown --out <dir>")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	result, err := exporter.Markdown(ctx, rt.Store, parseOutDir(args[1:]), parseLimit(args[1:], 100))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "out", result.OutDir)
	output.PrintKV(w, "files", result.Count)
	return nil
}
