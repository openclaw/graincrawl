package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/openclaw/crawlkit/control"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
	"github.com/openclaw/graincrawl/internal/store"
)

func (a App) runMetadata(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	_ = ctx
	cfg, configPath, err := config.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	manifest := controlManifest(configPath, cfg)
	if flags.JSON {
		return output.WriteJSON(w, manifest)
	}
	output.PrintKV(w, "id", manifest.ID)
	output.PrintKV(w, "binary", manifest.Binary.Name)
	output.PrintKV(w, "config", manifest.Paths.DefaultConfig)
	output.PrintKV(w, "database", manifest.Paths.DefaultDatabase)
	return nil
}

func controlManifest(configPath string, cfg config.Config) control.Manifest {
	manifest := control.NewManifest("graincrawl", "Granola Archive", "graincrawl")
	manifest.Description = "Local-first archive for Granola notes, transcripts, summaries, and panels."
	manifest.Branding = control.Branding{SymbolName: "note.text", AccentColor: "#d4a017", BundleIdentifier: "com.vincentkoc.graincrawl"}
	manifest.Paths = control.Paths{
		DefaultConfig:   configPath,
		ConfigEnv:       config.ConfigEnv,
		DefaultDatabase: cfg.Paths.DBPath,
		DefaultCache:    cfg.Paths.CacheDir,
		DefaultLogs:     cfg.Paths.LogDir,
	}
	manifest.Capabilities = []string{"metadata", "status", "doctor", "sync", "notes", "sql", "export", "snapshot", "tui"}
	manifest.Commands = map[string]control.Command{
		"metadata":     {Title: "Metadata", Argv: []string{"graincrawl", "metadata", "--json"}, JSON: true},
		"status":       {Title: "Status", Argv: []string{"graincrawl", "status", "--json"}, JSON: true},
		"check-update": {Title: "Check for updates", Argv: []string{"graincrawl", "check-update", "--json"}, JSON: true},
		"doctor":       {Title: "Doctor", Argv: []string{"graincrawl", "doctor", "--json"}, JSON: true},
		"sync":         {Title: "Sync", Argv: []string{"graincrawl", "sync", "--source", cfg.Granola.PreferredSource, "--json"}, JSON: true, Mutates: true},
		"notes":        {Title: "Notes", Argv: []string{"graincrawl", "notes", "--json"}, JSON: true},
		"sql":          {Title: "Read-only SQL", Argv: []string{"graincrawl", "--json", "sql", "select count(*) as notes from notes"}, JSON: true},
		"tui":          {Title: "TUI", Argv: []string{"graincrawl", "tui"}},
		"snapshot":     {Title: "Snapshot", Argv: []string{"graincrawl", "snapshot", "create"}, Mutates: true},
		"export":       {Title: "Markdown Export", Argv: []string{"graincrawl", "export", "markdown", "--out", "./granola-notes"}, Mutates: true},
		"unlock":       {Title: "Unlock", Argv: []string{"graincrawl", "unlock", "--json"}, JSON: true},
		"completion":   {Title: "Completion", Argv: []string{"graincrawl", "completion", "zsh"}},
		"legacy-json":  {Title: "Legacy JSON envelope", Argv: []string{"graincrawl", "version", "--json"}, JSON: true, Legacy: true},
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes:         []string{"Granola profile", "graincrawl SQLite archive"},
	}
	return manifest
}

func controlStatus(configPath string, cfg config.Config, status store.Status) control.Status {
	counts := []control.Count{
		control.NewCount("notes", "Notes", status.Notes),
		control.NewCount("transcripts", "Transcript chunks", status.Transcripts),
		control.NewCount("panels", "Panels", status.Panels),
		control.NewCount("source_objects", "Source objects", status.Sources),
		control.NewCount("sync_runs", "Sync runs", status.SyncRuns),
	}
	out := control.NewStatus("graincrawl", fmt.Sprintf("%d notes, %d transcript chunks, %d panels", status.Notes, status.Transcripts, status.Panels))
	out.State = "ok"
	out.ConfigPath = configPath
	out.DatabasePath = status.DBPath
	out.Counts = counts
	out.DatabaseBytes = fileSize(status.DBPath)
	out.WALBytes = fileSize(status.DBPath + "-wal")
	out.Databases = []control.Database{
		control.SQLiteDatabase("primary", "Granola archive", "archive", status.DBPath, true, counts),
	}
	if !status.LastSyncAt.IsZero() {
		out.LastSyncAt = status.LastSyncAt.UTC().Format(time.RFC3339)
	}
	if cfg.Granola.AllowEncryptedJSON || cfg.Granola.AllowOPFS {
		out.Warnings = append(out.Warnings, "encrypted local sources require explicit unlock")
	}
	return out
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func openRuntimeStatus(ctx context.Context, configPath string) (gruntime.Runtime, store.Status, error) {
	rt, err := gruntime.Open(ctx, configPath)
	if err != nil {
		return gruntime.Runtime{}, store.Status{}, err
	}
	status, err := rt.Store.Status(ctx)
	if err != nil {
		_ = rt.Close()
		return gruntime.Runtime{}, store.Status{}, err
	}
	return rt, status, nil
}
