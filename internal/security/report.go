package security

import (
	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/model"
)

type SourceSupport struct {
	Source      model.Source `json:"source"`
	Allowed     bool         `json:"allowed"`
	Implemented bool         `json:"implemented"`
	NeedsSecret bool         `json:"needs_secret"`
	Notes       string       `json:"notes,omitempty"`
}

type UnlockReport struct {
	KeychainPromptMode string `json:"keychain_prompt_mode"`
	PersistHelperKeys  bool   `json:"persist_helper_keys"`
	EncryptedJSON      bool   `json:"encrypted_json_enabled"`
	OPFS               bool   `json:"opfs_enabled"`
	RequiresCompanion  bool   `json:"requires_companion"`
	PromptAllowed      bool   `json:"prompt_allowed"`
	Message            string `json:"message"`
}

type SecretReport struct {
	ManagedSecrets     bool   `json:"managed_secrets"`
	PersistHelperKeys  bool   `json:"persist_helper_keys"`
	KeychainPromptMode string `json:"keychain_prompt_mode"`
	GranolaKeychain    string `json:"granola_keychain"`
	Message            string `json:"message"`
}

func Sources(cfg config.Config) []SourceSupport {
	return []SourceSupport{
		{
			Source:      model.SourcePrivateAPI,
			Allowed:     cfg.Granola.AllowPrivateAPI,
			Implemented: true,
			NeedsSecret: true,
			Notes:       "uses Granola desktop WorkOS token when present",
		},
		{
			Source:      model.SourceDesktopCache,
			Allowed:     cfg.Granola.AllowDesktopCache,
			Implemented: true,
			NeedsSecret: false,
			Notes:       "reads plaintext cache-v6.json when available",
		},
		{
			Source:      model.SourceEncryptedJSON,
			Allowed:     cfg.Granola.AllowEncryptedJSON,
			Implemented: false,
			NeedsSecret: true,
			Notes:       "unlock surface only; use --unlock encrypted-json with private-api or desktop-cache",
		},
		{
			Source:      model.SourceOPFS,
			Allowed:     cfg.Granola.AllowOPFS,
			Implemented: false,
			NeedsSecret: true,
			Notes:       "unsupported in this build; future explicit unlock/import must be security reviewed",
		},
		{
			Source:      model.SourcePublicAPI,
			Allowed:     cfg.Granola.AllowPublicAPI,
			Implemented: false,
			NeedsSecret: true,
			Notes:       "official API is currently limited compared with local archive goals",
		},
	}
}

func Unlock(cfg config.Config) UnlockReport {
	mode := cfg.Security.KeychainPromptMode
	return UnlockReport{
		KeychainPromptMode: mode,
		PersistHelperKeys:  cfg.Security.PersistHelperKeys,
		EncryptedJSON:      cfg.Granola.AllowEncryptedJSON,
		OPFS:               cfg.Granola.AllowOPFS,
		RequiresCompanion:  false,
		PromptAllowed:      PromptAllowed(mode),
		Message:            unlockMessage(cfg.Granola.AllowEncryptedJSON, cfg.Granola.AllowOPFS, mode),
	}
}

func PromptAllowed(mode string) bool {
	return mode == "explicit" || mode == "ask" || mode == "always"
}

func Secrets(cfg config.Config) SecretReport {
	return SecretReport{
		ManagedSecrets:     cfg.Security.PersistHelperKeys,
		PersistHelperKeys:  cfg.Security.PersistHelperKeys,
		KeychainPromptMode: cfg.Security.KeychainPromptMode,
		GranolaKeychain:    "external",
		Message:            "graincrawl does not persist Granola tokens; encrypted sources require an explicit unlock flow",
	}
}

func unlockMessage(encryptedJSON, opfs bool, mode string) string {
	if !encryptedJSON && !opfs {
		return "encrypted JSON and OPFS sources are disabled; no keychain prompt is expected"
	}
	if encryptedJSON && PromptAllowed(mode) {
		return "encrypted JSON unlock is available only through an explicit unlock command; OPFS remains unsupported"
	}
	if encryptedJSON {
		return "encrypted JSON is enabled, but the current keychain prompt mode blocks unlock; OPFS remains unsupported"
	}
	return "OPFS is configured but unsupported in this build; no companion or keychain prompt will run"
}
