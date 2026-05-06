package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vincentkoc/graincrawl/internal/config"
)

func TestAppStatusAndSecurityCommandsUseTempConfig(t *testing.T) {
	cfgPath := writeTestConfig(t)
	for _, command := range [][]string{
		{"--json", "--config", cfgPath, "status"},
		{"--json", "--config", cfgPath, "sources"},
		{"--json", "--config", cfgPath, "unlock"},
		{"--json", "--config", cfgPath, "secrets"},
	} {
		var out bytes.Buffer
		app := App{Stdout: &out}
		if err := app.Run(context.Background(), command); err != nil {
			t.Fatalf("%v failed: %v", command, err)
		}
		if !strings.Contains(out.String(), `"ok": true`) {
			t.Fatalf("%v did not return ok envelope: %s", command, out.String())
		}
	}
}

func TestAppRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(context.Background(), []string{"bogus"}); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cfg, _, err := config.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Paths.DBPath = filepath.Join(root, "graincrawl.db")
	cfg.Paths.CacheDir = filepath.Join(root, "cache")
	cfg.Paths.LogDir = filepath.Join(root, "logs")
	cfg.Paths.SnapshotDir = filepath.Join(root, "snapshots")
	cfg.Granola.ProfilePath = filepath.Join(root, "Granola")
	cfgPath := filepath.Join(root, "config.toml")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}
