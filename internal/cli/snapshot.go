package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
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
	path, replace, err := parseImportArgs(args)
	if err != nil {
		return err
	}
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	manifest, err := portable.Import(ctx, rt.Store, portable.Options{RootDir: path, Replace: replace})
	if err != nil {
		return err
	}
	mode := "merge"
	if replace {
		mode = "replace"
	}
	result := map[string]any{"snapshot_dir": path, "mode": mode, "manifest": manifest}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "imported", path)
	output.PrintKV(w, "mode", mode)
	output.PrintKV(w, "tables", len(manifest.Tables))
	return nil
}

func parseImportArgs(args []string) (string, bool, error) {
	var path string
	var replace bool
	for _, arg := range args {
		switch {
		case arg == "--replace":
			replace = true
		case strings.HasPrefix(arg, "-"):
			return "", false, fmt.Errorf("unknown import flag %q", arg)
		case path == "":
			path = arg
		default:
			return "", false, fmt.Errorf("usage: graincrawl import [--replace] <snapshot-dir>")
		}
	}
	if path == "" {
		return "", false, fmt.Errorf("usage: graincrawl import [--replace] <snapshot-dir>")
	}
	return path, replace, nil
}
