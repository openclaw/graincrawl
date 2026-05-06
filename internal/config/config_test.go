package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultsUseGraincrawlPaths(t *testing.T) {
	cfg, path, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 {
		t.Fatalf("version = %d", cfg.Version)
	}
	if filepath.Base(path) != "config.toml" {
		t.Fatalf("config path = %s", path)
	}
	if cfg.Granola.PreferredSource != "private-api" {
		t.Fatalf("source = %s", cfg.Granola.PreferredSource)
	}
	if !cfg.Sync.IncludeTranscripts || !cfg.Sync.IncludePanels {
		t.Fatalf("expected transcripts and panels enabled")
	}
}
