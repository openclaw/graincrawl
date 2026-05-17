package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/openclaw/graincrawl/internal/output"
	"github.com/openclaw/graincrawl/internal/portable"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
)

func (a App) runSnapshot(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return fmt.Errorf("usage: graincrawl snapshot create [--out <dir>]")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	outDir := parseOutDir(args[1:])
	if outDir == "" {
		outDir = portable.DefaultDir(rt.Config.Paths.SnapshotDir, time.Now())
	}
	manifest, err := portable.Export(ctx, rt.Store, portable.Options{RootDir: outDir})
	if err != nil {
		return err
	}
	result := map[string]any{"snapshot_dir": outDir, "manifest": manifest}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "snapshot", outDir)
	output.PrintKV(w, "tables", len(manifest.Tables))
	return nil
}

func (a App) runImport(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	path := importPath(args)
	if path == "" {
		return fmt.Errorf("usage: graincrawl import <snapshot-dir>")
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	manifest, err := portable.Import(ctx, rt.Store, portable.Options{RootDir: path})
	if err != nil {
		return err
	}
	result := map[string]any{"snapshot_dir": path, "manifest": manifest}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "imported", path)
	output.PrintKV(w, "tables", len(manifest.Tables))
	return nil
}

func importPath(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if args[0] == "snapshot" || args[0] == "import" {
		if len(args) > 1 {
			return args[1]
		}
		return ""
	}
	return args[0]
}
