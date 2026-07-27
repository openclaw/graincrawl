package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
)

func TestUnlockReportDoesNotRequireCompanion(t *testing.T) {
	cfg := config.Config{
		Granola:  config.GranolaConfig{AllowEncryptedJSON: true},
		Security: config.SecurityConfig{KeychainPromptMode: "explicit"},
	}
	report := Unlock(cfg, false)
	if report.RequiresCompanion {
		t.Fatal("in-process encrypted JSON unlock must not require a companion")
	}
	if !report.PromptAllowed || !strings.Contains(report.Message, "explicit unlock") {
		t.Fatalf("unexpected encrypted JSON unlock report: %#v", report)
	}
}

func TestUnlockReportExplainsUnsupportedOPFSWithoutCompanion(t *testing.T) {
	report := Unlock(config.Config{Granola: config.GranolaConfig{AllowOPFS: true}}, false)
	if report.RequiresCompanion {
		t.Fatal("unsupported OPFS must not claim a running companion requirement")
	}
	if !strings.Contains(report.Message, "unsupported") || !strings.Contains(report.Message, "no companion") {
		t.Fatalf("unexpected OPFS unlock report: %#v", report)
	}
}

func TestMigratedProfileBlocksLocalSourcesAndUnlock(t *testing.T) {
	cfg := config.Config{
		Granola: config.GranolaConfig{
			AllowPrivateAPI:    true,
			AllowDesktopCache:  true,
			AllowEncryptedJSON: true,
		},
		Security: config.SecurityConfig{KeychainPromptMode: "explicit"},
	}
	profile := t.TempDir()
	for _, name := range []string{"cache-v6.json.enc", "supabase.json.enc"} {
		if err := os.WriteFile(filepath.Join(profile, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths := granola.Paths(profile, "")
	for _, source := range Sources(cfg, paths) {
		switch source.Source {
		case "private-api", "desktop-cache", "encrypted-json":
			if source.Allowed || source.Notes != granola.PostMigrationStateMessage {
				t.Fatalf("migrated source support = %#v", source)
			}
		}
	}

	// A migrated profile whose plaintext cache is still usable must keep
	// desktop-cache advertised, matching sync behavior.
	mixed := t.TempDir()
	if err := os.WriteFile(filepath.Join(mixed, "supabase.json.enc"), []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mixed, "cache-v6.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, source := range Sources(cfg, granola.Paths(mixed, "")) {
		switch source.Source {
		case "desktop-cache":
			if !source.Allowed {
				t.Fatalf("usable plaintext cache reported unavailable: %#v", source)
			}
		case "private-api":
			if source.Allowed {
				t.Fatalf("encrypted supabase state should block private-api: %#v", source)
			}
		}
	}

	report := Unlock(cfg, true)
	if report.PromptAllowed || report.Message != granola.PostMigrationStateMessage {
		t.Fatalf("migrated unlock report = %#v", report)
	}
}
