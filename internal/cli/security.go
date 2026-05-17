package cli

import (
	"context"
	"io"

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

func (a App) runUnlock(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := gruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
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
