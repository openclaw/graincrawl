package security

import (
	"strings"
	"testing"

	"github.com/openclaw/graincrawl/internal/config"
)

func TestUnlockReportDoesNotRequireCompanion(t *testing.T) {
	cfg := config.Config{
		Granola:  config.GranolaConfig{AllowEncryptedJSON: true},
		Security: config.SecurityConfig{KeychainPromptMode: "explicit"},
	}
	report := Unlock(cfg)
	if report.RequiresCompanion {
		t.Fatal("in-process encrypted JSON unlock must not require a companion")
	}
	if !report.PromptAllowed || !strings.Contains(report.Message, "explicit unlock") {
		t.Fatalf("unexpected encrypted JSON unlock report: %#v", report)
	}
}

func TestUnlockReportExplainsUnsupportedOPFSWithoutCompanion(t *testing.T) {
	report := Unlock(config.Config{Granola: config.GranolaConfig{AllowOPFS: true}})
	if report.RequiresCompanion {
		t.Fatal("unsupported OPFS must not claim a running companion requirement")
	}
	if !strings.Contains(report.Message, "unsupported") || !strings.Contains(report.Message, "no companion") {
		t.Fatalf("unexpected OPFS unlock report: %#v", report)
	}
}
