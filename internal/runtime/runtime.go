package runtime

import (
	"context"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/store"
)

type Runtime struct {
	Config     config.Config
	ConfigPath string
	Store      *store.Store
}

func Open(ctx context.Context, configPath string) (Runtime, error) {
	cfg, resolved, err := config.Load(configPath)
	if err != nil {
		return Runtime{}, err
	}
	if err := config.EnsureDirs(cfg); err != nil {
		return Runtime{}, err
	}
	st, err := store.Open(ctx, cfg.Paths.DBPath)
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{Config: cfg, ConfigPath: resolved, Store: st}, nil
}

func (r Runtime) Close() error {
	if r.Store == nil {
		return nil
	}
	return r.Store.Close()
}
