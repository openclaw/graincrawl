package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/openclaw/crawlkit/releasecheck"
	"github.com/openclaw/graincrawl/internal/buildinfo"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/output"
)

const graincrawlUpgradeHint = "brew upgrade openclaw/tap/graincrawl"

func graincrawlReleaseCheckOptions(force bool) releasecheck.Options {
	cfg, _, err := config.Defaults()
	cacheDir := ""
	if err == nil {
		cacheDir = cfg.Paths.CacheDir
	}
	return releasecheck.Options{
		AppName:        "graincrawl",
		Owner:          "openclaw",
		Repo:           "graincrawl",
		CurrentVersion: buildinfo.Current().Version,
		CacheDir:       cacheDir,
		Force:          force,
	}
}

func (a App) maybeNotifyRelease(ctx context.Context, args []string, flags GlobalFlags) {
	_, _ = releasecheck.Notify(ctx, releasecheck.NotifyOptions{
		Options:     graincrawlReleaseCheckOptions(false),
		Stderr:      a.Stderr,
		InstallHint: graincrawlUpgradeHint,
		Args:        args,
		JSONOutput:  flags.JSON,
		IsTerminal:  releasecheck.StderrIsTerminal(),
	})
}

func (a App) runCheckUpdate(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	fs := flag.NewFlagSet("check-update", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	jsonOut := fs.Bool("json", false, "write JSON output")
	force := fs.Bool("force", false, "force a fresh release check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("check-update takes flags only")
	}
	result, err := releasecheck.Check(ctx, graincrawlReleaseCheckOptions(*force))
	if err != nil && !errors.Is(err, releasecheck.ErrSkipped) {
		return err
	}
	if flags.JSON || *jsonOut {
		return output.WriteEnvelope(w, result)
	}
	_, err = fmt.Fprint(w, releasecheck.StatusText("graincrawl", graincrawlUpgradeHint, result))
	return err
}
