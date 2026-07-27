package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
)

func TestRunReportsEncryptedOnlyGranolaState(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"supabase.json.enc", "cache-v6.json.enc", "storage.dek"} {
		if err := os.WriteFile(filepath.Join(profile, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	report := Run(config.Config{
		Granola: config.GranolaConfig{ProfilePath: profile},
	}, "/tmp/graincrawl.toml", time.Now())
	if !report.Unlock.EncryptedJSONRequired {
		t.Fatal("expected encrypted JSON to be required")
	}
	if report.Unlock.KeychainMayPrompt {
		t.Fatal("diagnostic-only encrypted state must not imply a keychain prompt")
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.Code != "granola_encrypted_only_state" || diagnostic.Severity != "warning" {
		t.Fatalf("unexpected diagnostic metadata: %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "encrypted-only") ||
		!strings.Contains(diagnostic.Message, "explicit") ||
		!strings.Contains(diagnostic.Message, "supabase.json") {
		t.Fatalf("diagnostic does not explain encrypted-only state: %q", diagnostic.Message)
	}
	if diagnostic.Message != granola.EncryptedOnlyStateMessage {
		t.Fatalf("doctor should use shared message, got %q", diagnostic.Message)
	}
}

func TestRunReportsPostMigrationStateWithGranolaVersion(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "Granola")
	appPath := filepath.Join(root, "Granola.app")
	if err := os.MkdirAll(filepath.Join(appPath, "Contents"), 0o700); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0"?><plist><dict><key>CFBundleShortVersionString</key><string>7.441.6</string><key>CFBundleIdentifier</key><string>com.granola.app</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), []byte(plist), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "stored-accounts.json.enc"), []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}

	report := Run(config.Config{
		Granola: config.GranolaConfig{ProfilePath: profile, AppPath: appPath},
	}, "/tmp/graincrawl.toml", time.Now())
	if !report.Unlock.PostMigrationState {
		t.Fatal("expected post-migration state")
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.Code != "granola_dek_keychain_migration" || diagnostic.Severity != "error" {
		t.Fatalf("unexpected diagnostic metadata: %#v", diagnostic)
	}
	want := granola.PostMigrationStateMessage + " Detected Granola version 7.441.6."
	if diagnostic.Message != want {
		t.Fatalf("diagnostic = %q, want %q", diagnostic.Message, want)
	}
}

func TestRunReportsEncryptedOnlyCacheState(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "Granola")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "cache-v6.json.enc"), []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "storage.dek"), []byte("encrypted"), 0o600); err != nil {
		t.Fatal(err)
	}
	supabaseRaw := `{"workos_tokens":"{\"access_token\":\"token\",\"obtained_at\":4102444800000,\"expires_in\":3600}"}`
	if err := os.WriteFile(filepath.Join(profile, "supabase.json"), []byte(supabaseRaw), 0o600); err != nil {
		t.Fatal(err)
	}

	report := Run(config.Config{
		Granola: config.GranolaConfig{ProfilePath: profile},
	}, "/tmp/graincrawl.toml", time.Now())
	if len(report.Diagnostics) != 1 {
		t.Fatalf("expected encrypted-cache diagnostic, got %#v", report.Diagnostics)
	}
	if !strings.Contains(report.Diagnostics[0].Message, "cache-v6.json") {
		t.Fatalf("diagnostic should mention plaintext cache, got %q", report.Diagnostics[0].Message)
	}
}
