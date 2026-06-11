package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/openclaw/graincrawl/internal/cachev6"
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/encryptedjson"
	"github.com/openclaw/graincrawl/internal/granola"
	"github.com/openclaw/graincrawl/internal/output"
	gruntime "github.com/openclaw/graincrawl/internal/runtime"
	"github.com/openclaw/graincrawl/internal/security"
)

func (a App) runSources(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	sources := security.Sources(rt.Config)
	if flags.JSON {
		return output.WriteEnvelope(w, map[string]any{"sources": sources})
	}
	for _, source := range sources {
		output.PrintKV(w, string(source.Source), source.Allowed)
	}
	return nil
}

type encryptedJSONUnlockResult struct {
	Surface          string                `json:"surface"`
	KeychainAccessed bool                  `json:"keychain_accessed"`
	Files            []string              `json:"files"`
	Cache            *cachev6.Summary      `json:"cache,omitempty"`
	Token            *granola.TokenSummary `json:"token,omitempty"`
}

func (a App) runUnlock(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	if len(args) > 0 {
		if len(args) != 1 || args[0] != "encrypted-json" {
			return fmt.Errorf("unsupported unlock surface %q", args[0])
		}
		return a.runEncryptedJSONUnlock(ctx, w, flags, rt.Config)
	}
	report := security.Unlock(rt.Config)
	if flags.JSON {
		return output.WriteEnvelope(w, report)
	}
	output.PrintKV(w, "keychain_prompt_mode", report.KeychainPromptMode)
	output.PrintKV(w, "prompt_allowed", report.PromptAllowed)
	output.PrintKV(w, "requires_companion", report.RequiresCompanion)
	output.PrintKV(w, "message", report.Message)
	return nil
}

func (a App) runEncryptedJSONUnlock(ctx context.Context, w io.Writer, flags GlobalFlags, cfg config.Config) error {
	if !cfg.Granola.AllowEncryptedJSON {
		return fmt.Errorf("encrypted-json source disabled in config")
	}
	if !security.PromptAllowed(cfg.Security.KeychainPromptMode) {
		return fmt.Errorf("keychain prompt mode %q blocks encrypted-json unlock", cfg.Security.KeychainPromptMode)
	}
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	names := make([]string, 0, 2)
	if _, err := os.Stat(paths.CacheV6Encrypted); err == nil {
		names = append(names, encryptedjson.CacheFile)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect encrypted cache: %w", err)
	}
	if _, err := os.Stat(paths.SupabaseEncrypted); err == nil {
		names = append(names, encryptedjson.SupabaseFile)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect encrypted auth state: %w", err)
	}
	if len(names) == 0 {
		return fmt.Errorf("no encrypted Granola JSON files found")
	}
	files, err := a.encryptedJSONDecryptor()(ctx, paths.Root, names...)
	if err != nil {
		return err
	}
	defer encryptedjson.Clear(files)
	result := encryptedJSONUnlockResult{
		Surface:          "encrypted-json",
		KeychainAccessed: true,
		Files:            names,
	}
	if raw, ok := files[encryptedjson.CacheFile]; ok {
		file, err := cachev6.Parse(raw)
		if err != nil {
			return fmt.Errorf("parse decrypted cache: %w", err)
		}
		summary := cachev6.Summarize(file)
		result.Cache = &summary
	}
	if raw, ok := files[encryptedjson.SupabaseFile]; ok {
		_, tokens, _, err := granola.ParseSupabase(raw)
		if err != nil {
			return fmt.Errorf("parse decrypted auth state: %w", err)
		}
		summary := granola.SummarizeToken(tokens, time.Now())
		result.Token = &summary
	}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "surface", result.Surface)
	output.PrintKV(w, "keychain_accessed", result.KeychainAccessed)
	if result.Cache != nil {
		output.PrintKV(w, "cache_version", result.Cache.Version)
		output.PrintKV(w, "cache_documents", result.Cache.DocumentCount)
	}
	if result.Token != nil {
		output.PrintKV(w, "token_present", result.Token.Present)
		output.PrintKV(w, "token_expired", result.Token.Expired)
	}
	return nil
}

func (a App) runSecrets(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	report := security.Secrets(rt.Config)
	if flags.JSON {
		return output.WriteEnvelope(w, report)
	}
	output.PrintKV(w, "managed_secrets", report.ManagedSecrets)
	output.PrintKV(w, "persist_helper_keys", report.PersistHelperKeys)
	output.PrintKV(w, "keychain_prompt_mode", report.KeychainPromptMode)
	output.PrintKV(w, "granola_keychain", report.GranolaKeychain)
	output.PrintKV(w, "message", report.Message)
	return nil
}
