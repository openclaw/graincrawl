package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/openclaw/graincrawl/internal/buildinfo"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/doctor"
	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
	"github.com/openclaw/graincrawl/internal/syncer"
)

type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (a App) Run(ctx context.Context, args []string) error {
	stdout := a.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	flags, rest := parseGlobalFlags(args)
	if flags.Version {
		return a.runVersion(stdout, flags)
	}
	if flags.Help || len(rest) == 0 {
		_, err := io.WriteString(stdout, usage)
		return err
	}
	cmd, cmdArgs := rest[0], rest[1:]
	a.maybeNotifyRelease(ctx, rest, flags)
	switch cmd {
	case "version":
		return a.runVersion(stdout, flags)
	case "check-update":
		return a.runCheckUpdate(ctx, stdout, flags, cmdArgs)
	case "init":
		return a.runInit(stdout, flags)
	case "doctor":
		return a.runDoctor(ctx, stdout, flags)
	case "metadata":
		return a.runMetadata(ctx, stdout, flags)
	case "sync":
		return a.runSync(ctx, stdout, flags, cmdArgs)
	case "refresh":
		return a.runSync(ctx, stdout, flags, cmdArgs)
	case "status":
		return a.runStatus(ctx, stdout, flags)
	case "runs":
		return a.runRuns(ctx, stdout, flags, cmdArgs)
	case "notes":
		return a.runNotes(ctx, stdout, flags, cmdArgs)
	case "search":
		return a.runSearch(ctx, stdout, flags, cmdArgs)
	case "sql":
		return a.runSQL(ctx, stdout, flags, cmdArgs)
	case "note":
		return a.runNote(ctx, stdout, flags, cmdArgs)
	case "transcripts":
		return a.runTranscripts(ctx, stdout, flags, cmdArgs)
	case "panels":
		return a.runPanels(ctx, stdout, flags, cmdArgs)
	case "people":
		return a.runSourceObjectList(ctx, stdout, flags, cmdArgs, "person")
	case "workspaces":
		return a.runSourceObjectList(ctx, stdout, flags, cmdArgs, "workspace")
	case "sources":
		return a.runSources(ctx, stdout, flags)
	case "unlock":
		return a.runUnlock(ctx, stdout, flags)
	case "secrets":
		return a.runSecrets(ctx, stdout, flags)
	case "export":
		return a.runExport(ctx, stdout, flags, cmdArgs)
	case "snapshot":
		return a.runSnapshot(ctx, stdout, flags, cmdArgs)
	case "import":
		return a.runImport(ctx, stdout, flags, cmdArgs)
	case "tui":
		return a.runTUI(ctx, stdout, flags, cmdArgs)
	case "completion":
		return a.runCompletion(stdout, flags, cmdArgs)
	case "help":
		_, err := io.WriteString(stdout, usage)
		return err
	default:
		_ = ctx
		_ = cmdArgs
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func (a App) runSync(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	result, err := syncer.Run(ctx, rt.Config, rt.Store, parseSyncOptions(args))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "source", result.Source)
	output.PrintKV(w, "notes", result.Notes)
	output.PrintKV(w, "transcripts", result.Transcripts)
	output.PrintKV(w, "panels", result.Panels)
	return nil
}

func (a App) runStatus(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	status, err := rt.Store.Status(ctx)
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.WriteJSON(w, controlStatus(rt.ConfigPath, rt.Config, status))
	}
	output.PrintKV(w, "database", rt.Store.Path())
	output.PrintKV(w, "notes", status.Notes)
	output.PrintKV(w, "transcripts", status.Transcripts)
	output.PrintKV(w, "panels", status.Panels)
	output.PrintKV(w, "sync_runs", status.SyncRuns)
	return nil
}

func (a App) runInit(w io.Writer, flags GlobalFlags) error {
	cfg, defaultPath, err := config.Defaults()
	if err != nil {
		return err
	}
	path, err := config.App().ResolveConfigPath(flags.ConfigPath)
	if err != nil {
		return err
	}
	if path == "" {
		path = defaultPath
	}
	if err := config.EnsureDirs(cfg); err != nil {
		return err
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	result := map[string]string{"config_path": path, "db_path": cfg.Paths.DBPath}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "config", path)
	output.PrintKV(w, "database", cfg.Paths.DBPath)
	return nil
}

func (a App) runDoctor(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	report := doctor.Run(rt.Config, rt.ConfigPath, time.Now())
	if flags.JSON {
		return output.WriteEnvelope(w, report)
	}
	output.PrintKV(w, "config", report.ConfigPath)
	output.PrintKV(w, "database", report.DBPath)
	output.PrintKV(w, "granola_app", report.GranolaApp.Installed)
	output.PrintKV(w, "granola_version", report.GranolaApp.Version)
	output.PrintKV(w, "cache_v6", report.Files.CacheV6.Exists)
	output.PrintKV(w, "supabase", report.Files.Supabase.Exists)
	output.PrintKV(w, "opfs_present", report.Unlock.OPFSPresent)
	output.PrintKV(w, "keychain_may_prompt", report.Unlock.KeychainMayPrompt)
	return nil
}

func (a App) runVersion(w io.Writer, flags GlobalFlags) error {
	info := buildinfo.Current()
	if flags.JSON {
		return output.WriteEnvelope(w, info)
	}
	output.PrintKV(w, "version", info.Version)
	output.PrintKV(w, "commit", info.Commit)
	output.PrintKV(w, "date", info.Date)
	return nil
}
