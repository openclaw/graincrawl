package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/store"
)

func TestAppStatusAndSecurityCommandsUseTempConfig(t *testing.T) {
	cfgPath := writeTestConfig(t)
	for _, command := range [][]string{
		{"--json", "--config", cfgPath, "sources"},
		{"--json", "--config", cfgPath, "unlock"},
		{"--json", "--config", cfgPath, "secrets"},
		{"--json", "--config", cfgPath, "completion"},
		{"--json", "--config", cfgPath, "runs"},
		{"--json", "--config", cfgPath, "people"},
		{"--json", "--config", cfgPath, "workspaces"},
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
	var tuiOut bytes.Buffer
	app := App{Stdout: &tuiOut}
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "tui"}); err != nil {
		t.Fatalf("tui json failed: %v", err)
	}
	if !strings.Contains(tuiOut.String(), "[]") {
		t.Fatalf("tui json did not return rows: %s", tuiOut.String())
	}
	for _, command := range [][]string{
		{"--json", "--config", cfgPath, "metadata"},
		{"--json", "--config", cfgPath, "status"},
	} {
		var out bytes.Buffer
		app := App{Stdout: &out}
		if err := app.Run(context.Background(), command); err != nil {
			t.Fatalf("%v failed: %v", command, err)
		}
		if !strings.Contains(out.String(), `"schema_version": "crawlkit.control.v1"`) {
			t.Fatalf("%v did not return crawlkit control JSON: %s", command, out.String())
		}
	}
	var metadataOut bytes.Buffer
	metadataApp := App{Stdout: &metadataOut}
	if err := metadataApp.Run(context.Background(), []string{"--json", "--config", cfgPath, "metadata"}); err != nil {
		t.Fatalf("metadata failed: %v", err)
	}
	if !strings.Contains(metadataOut.String(), `"desktop-cache-import"`) ||
		!strings.Contains(metadataOut.String(), `"desktop-cache"`) {
		t.Fatalf("metadata did not expose desktop cache import: %s", metadataOut.String())
	}
}

func TestAppGlobalVersionFlag(t *testing.T) {
	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if !strings.Contains(out.String(), "version") {
		t.Fatalf("--version output missing version: %s", out.String())
	}
}

func TestAppReportsEncryptedOnlyGranolaState(t *testing.T) {
	cfgPath := writeTestConfig(t)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.Granola.ProfilePath, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheRaw := `{"cache":{"version":8,"state":{"transcripts":{}}}}`
	if err := os.WriteFile(filepath.Join(cfg.Granola.ProfilePath, "cache-v6.json"), []byte(cacheRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"supabase.json.enc", "storage.dek"} {
		if err := os.WriteFile(filepath.Join(cfg.Granola.ProfilePath, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	for _, command := range [][]string{
		{"--json", "--config", cfgPath, "doctor"},
		{"--config", cfgPath, "doctor"},
		{"--json", "--config", cfgPath, "sync"},
		{"--json", "--config", cfgPath, "sync", "--source", "desktop-cache"},
		{"--config", cfgPath, "sync", "--source", "desktop-cache"},
	} {
		var out bytes.Buffer
		app := App{Stdout: &out}
		if err := app.Run(context.Background(), command); err != nil {
			t.Fatalf("%v failed: %v", command, err)
		}
		text := out.String()
		for _, want := range []string{"encrypted-only", "explicit", "supabase.json"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%v output missing %q:\n%s", command, want, text)
			}
		}
	}

	var unlockOut bytes.Buffer
	app := App{Stdout: &unlockOut}
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "unlock"}); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if !strings.Contains(unlockOut.String(), "disabled") || strings.Contains(unlockOut.String(), "keychain_accessed") {
		t.Fatalf("unlock status should remain prompt-free when disabled:\n%s", unlockOut.String())
	}
}

func TestAppExplicitEncryptedJSONUnlockRedactsPayload(t *testing.T) {
	cfgPath := writeTestConfig(t)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Granola.AllowEncryptedJSON = true
	cfg.Security.KeychainPromptMode = "explicit"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.Granola.ProfilePath, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"storage.dek", "cache-v6.json.enc", "supabase.json.enc"} {
		if err := os.WriteFile(filepath.Join(cfg.Granola.ProfilePath, name), []byte("encrypted"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cacheRaw := json.RawMessage(`{"cache":{"version":8,"state":{"documents":{"doc1":{"id":"doc1","created_at":"2026-05-06T01:00:00Z","updated_at":"2026-05-06T02:00:00Z","title":"Sensitive title","notes_plain":"Sensitive note"}},"transcripts":{}}}}`)
	supabaseRaw := json.RawMessage(`{"workos_tokens":"{\"access_token\":\"synthetic-secret-token\",\"refresh_token\":\"synthetic-refresh-token\",\"obtained_at\":4102444800000,\"expires_in\":3600}","user_info":"{\"email\":\"private@example.com\"}"}`)
	decryptCalls := 0
	app := App{
		DecryptEncryptedJSON: func(_ context.Context, _ string, names ...string) (map[string]json.RawMessage, error) {
			decryptCalls++
			return map[string]json.RawMessage{
				"cache-v6.json.enc": cacheRaw,
				"supabase.json.enc": supabaseRaw,
			}, nil
		},
	}

	var doctorOut bytes.Buffer
	app.Stdout = &doctorOut
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "doctor"}); err != nil {
		t.Fatal(err)
	}
	if decryptCalls != 0 {
		t.Fatal("doctor invoked encrypted-json decryptor")
	}

	var unlockOut bytes.Buffer
	app.Stdout = &unlockOut
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "unlock", "encrypted-json"}); err != nil {
		t.Fatal(err)
	}
	if decryptCalls != 1 {
		t.Fatalf("expected one explicit decryptor call, got %d", decryptCalls)
	}
	text := unlockOut.String()
	for _, want := range []string{`"surface": "encrypted-json"`, `"document_count": 1`, `"present": true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("unlock output missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"synthetic-secret-token", "synthetic-refresh-token", "Sensitive title", "Sensitive note", "private@example.com"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unlock output leaked %q:\n%s", forbidden, text)
		}
	}
}

func TestAppFormerEncryptedJSONHelperCommandCannotDecrypt(t *testing.T) {
	decryptCalls := 0
	var out bytes.Buffer
	app := App{
		Stdout: &out,
		DecryptEncryptedJSON: func(context.Context, string, ...string) (map[string]json.RawMessage, error) {
			decryptCalls++
			return nil, nil
		},
	}
	err := app.Run(context.Background(), []string{"_encrypted-json-helper"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected former helper command rejection, got %v", err)
	}
	if decryptCalls != 0 {
		t.Fatalf("former helper command invoked decryptor %d times", decryptCalls)
	}
	if out.Len() != 0 {
		t.Fatalf("former helper command wrote output: %q", out.String())
	}
}

func TestAppSnapshotExportImportUseTempArchive(t *testing.T) {
	cfgPath := writeTestConfig(t)
	snapshotDir := filepath.Join(t.TempDir(), "snapshot")
	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "snapshot", "create", "--out", snapshotDir}); err != nil {
		t.Fatalf("snapshot create failed: %v", err)
	}
	if !strings.Contains(out.String(), `"manifest"`) {
		t.Fatalf("snapshot output missing manifest: %s", out.String())
	}
	out.Reset()
	if err := app.Run(context.Background(), []string{"--json", "--config", cfgPath, "import", snapshotDir}); err != nil {
		t.Fatalf("snapshot import failed: %v", err)
	}
	if !strings.Contains(out.String(), `"manifest"`) {
		t.Fatalf("import output missing manifest: %s", out.String())
	}
}

func TestTUIJSONIncludesNoteTranscriptAndPanelDetails(t *testing.T) {
	ctx := context.Background()
	cfgPath := writeTestConfig(t)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, cfg.Paths.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	title := "Product Review"
	noteText := "note body decision"
	panelText := "panel action item"
	if err := st.UpsertNote(ctx, model.Note{
		ID:            "doc-1",
		Title:         &title,
		Type:          "meeting",
		CreatedAt:     now,
		UpdatedAt:     now,
		NotesMarkdown: &noteText,
		Source:        model.SourcePrivateAPI,
		LastSeenAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTranscriptChunk(ctx, model.TranscriptChunk{
		ID:             "chunk-1",
		DocumentID:     "doc-1",
		StartTimestamp: now,
		EndTimestamp:   now.Add(time.Second),
		Source:         "mic",
		Text:           "conversation transcript text",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertPanel(ctx, model.Panel{
		ID:              "panel-1",
		DocumentID:      "doc-1",
		Title:           &title,
		ContentMarkdown: &panelText,
		CreatedAt:       now,
		Source:          model.SourcePrivateAPI,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(ctx, []string{"--json", "--config", cfgPath, "tui"}); err != nil {
		t.Fatalf("tui json failed: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatalf("parse tui rows: %v\n%s", err, out.String())
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 tui row, got %d", len(rows))
	}
	detail := rows[0]["detail"].(string)
	for _, want := range []string{"## Notes", "note body decision", "## Conversation", "conversation transcript text", "## Panels", "panel action item"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q:\n%s", want, detail)
		}
	}
}

func TestAppSQLRunsReadOnlyQueries(t *testing.T) {
	ctx := context.Background()
	cfgPath := writeTestConfig(t)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, cfg.Paths.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	title := "Planning"
	if err := st.UpsertNote(ctx, model.Note{
		ID:         "doc-1",
		Title:      &title,
		Type:       "meeting",
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     model.SourcePrivateAPI,
		LastSeenAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(ctx, []string{"--json", "--config", cfgPath, "sql", "select source, count(*) as notes from notes group by source"}); err != nil {
		t.Fatalf("sql failed: %v", err)
	}
	if !strings.Contains(out.String(), `"columns"`) || !strings.Contains(out.String(), `private-api`) {
		t.Fatalf("sql json missing result: %s", out.String())
	}
	if err := app.Run(ctx, []string{"--config", cfgPath, "sql", "delete from notes"}); err == nil || !strings.Contains(err.Error(), "only read-only sql is allowed") {
		t.Fatalf("expected read-only rejection, got %v", err)
	}
}

func TestAppRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	app := App{Stdout: &out}
	if err := app.Run(context.Background(), []string{"bogus"}); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func TestParseSyncOptionsKeepsSkipFlags(t *testing.T) {
	opts := parseSyncOptions([]string{"--source", "desktop-cache", "--limit", "2", "--unlock", "encrypted-json", "--no-transcripts", "--no-panels"})
	if opts.Source != model.SourceDesktopCache || opts.Limit != 2 {
		t.Fatalf("bad source or limit: %#v", opts)
	}
	if opts.UnlockSurface != "encrypted-json" {
		t.Fatalf("unlock surface = %q", opts.UnlockSurface)
	}
	if !opts.SkipTranscripts || !opts.SkipPanels {
		t.Fatalf("expected skip flags, got %#v", opts)
	}
}

func TestShellCompletionsIncludeEncryptedJSONUnlock(t *testing.T) {
	for shell, completion := range map[string]string{
		"bash": bashCompletion(),
		"zsh":  zshCompletion(),
	} {
		for _, want := range []string{"--unlock", "encrypted-json"} {
			if !strings.Contains(completion, want) {
				t.Fatalf("%s completion missing %q", shell, want)
			}
		}
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
