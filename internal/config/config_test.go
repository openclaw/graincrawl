package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDefaultsAllowPublicAPIOnlyWhenExplicitlyEnabled(t *testing.T) {
	t.Setenv("GRAINCRAWL_ALLOW_PUBLIC_API", "true")
	cfg, _, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Granola.AllowPublicAPI {
		t.Fatal("expected public API source enabled from environment")
	}
}

func TestLoadPublicAPIEnvironmentOverridesConfigDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[granola]\nallow_public_api = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAINCRAWL_ALLOW_PUBLIC_API", "true")
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Granola.AllowPublicAPI {
		t.Fatal("expected explicit environment gate to override config default")
	}
}

func TestSaveOmitsTestOnlyPublicBaseURL(t *testing.T) {
	cfg, _, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	cfg.API.PublicBaseURL = "https://example.invalid"
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || strings.Contains(string(raw), "public_base_url") || strings.Contains(string(raw), "example.invalid") {
		t.Fatalf("test-only base URL was persisted:\n%s", raw)
	}
}
