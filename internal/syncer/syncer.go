package syncer

import (
	"context"
	"fmt"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func Run(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	if opts.Source == "" {
		opts.Source = model.Source(cfg.Granola.PreferredSource)
	}
	if opts.Limit == 0 {
		opts.Limit = cfg.Sync.DefaultLimit
	}
	opts.IncludeTranscripts = !opts.SkipTranscripts && (opts.IncludeTranscripts || cfg.Sync.IncludeTranscripts)
	opts.IncludePanels = !opts.SkipPanels && (opts.IncludePanels || cfg.Sync.IncludePanels)
	switch opts.Source {
	case model.SourcePrivateAPI:
		if !cfg.Granola.AllowPrivateAPI {
			return Result{}, fmt.Errorf("private-api source disabled in config")
		}
		return PrivateAPI(ctx, cfg, st, opts)
	case model.SourceDesktopCache:
		if !cfg.Granola.AllowDesktopCache {
			return Result{}, fmt.Errorf("desktop-cache source disabled in config")
		}
		return DesktopCache(ctx, cfg, st, opts)
	default:
		return Result{}, fmt.Errorf("source %q is disabled or unsupported in this build", opts.Source)
	}
}
