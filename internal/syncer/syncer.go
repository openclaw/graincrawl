package syncer

import (
	"context"
	"errors"
	"fmt"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/encryptedjson"
	"github.com/openclaw/graincrawl/internal/granola"
	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/security"
	"github.com/openclaw/graincrawl/internal/store"
)

func Run(ctx context.Context, cfg config.Config, st *store.Store, opts Options) (Result, error) {
	decrypt, err := encryptedJSONDecryptor(cfg, opts)
	if err != nil {
		return Result{}, err
	}
	sourceExplicit := opts.Source != ""
	if opts.Source == "" {
		opts.Source = model.Source(cfg.Granola.PreferredSource)
	}
	if opts.Limit == 0 {
		opts.Limit = cfg.Sync.DefaultLimit
	}
	opts.IncludeTranscripts = !opts.SkipTranscripts && (opts.IncludeTranscripts || cfg.Sync.IncludeTranscripts)
	opts.IncludePanels = !opts.SkipPanels && (opts.IncludePanels || cfg.Sync.IncludePanels)
	switch opts.Source {
	case model.SourcePrivateAPI:
		if !cfg.Granola.AllowPrivateAPI {
			return Result{}, fmt.Errorf("private-api source disabled in config")
		}
		encryptedSupabase := granola.EncryptedSupabaseState(granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath))
		if encryptedSupabase && decrypt == nil && !sourceExplicit && cfg.Granola.AllowDesktopCache {
			opts.Source = model.SourceDesktopCache
			return runDesktopCache(ctx, cfg, st, opts, decrypt)
		}
		if encryptedSupabase {
			if decrypt == nil {
				return Result{Source: model.SourcePrivateAPI, Message: granola.EncryptedOnlyStateMessage}, fmt.Errorf("private-api source requires plaintext supabase.json: %s", granola.EncryptedOnlyStateMessage)
			}
			paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
			names := []string{encryptedjson.SupabaseFile}
			encryptedCacheFallback := !sourceExplicit && cfg.Granola.AllowDesktopCache && granola.EncryptedCacheState(paths)
			if encryptedCacheFallback {
				names = append(names, encryptedjson.CacheFile)
			}
			files, err := decrypt(ctx, paths.Root, names...)
			if err != nil {
				return Result{Source: model.SourcePrivateAPI}, err
			}
			defer encryptedjson.Clear(files)
			raw, ok := files[encryptedjson.SupabaseFile]
			if !ok {
				return Result{Source: model.SourcePrivateAPI}, fmt.Errorf("encrypted storage decryptor omitted %s", encryptedjson.SupabaseFile)
			}
			result, err := privateAPIFromEncryptedSupabase(ctx, cfg, st, opts, raw)
			if err != nil && !sourceExplicit && cfg.Granola.AllowDesktopCache && privateAPIAuthUnavailable(err) {
				opts.Source = model.SourceDesktopCache
				if encryptedCacheFallback {
					cache, ok := files[encryptedjson.CacheFile]
					if !ok {
						return Result{Source: model.SourceEncryptedJSON}, fmt.Errorf("encrypted storage decryptor omitted %s", encryptedjson.CacheFile)
					}
					return encryptedDesktopCache(ctx, st, opts, cache)
				}
				return runDesktopCache(ctx, cfg, st, opts, nil)
			}
			return result, err
		}
		result, err := PrivateAPI(ctx, cfg, st, opts)
		if err != nil && !sourceExplicit && cfg.Granola.AllowDesktopCache && privateAPIAuthUnavailable(err) {
			opts.Source = model.SourceDesktopCache
			return runDesktopCache(ctx, cfg, st, opts, decrypt)
		}
		return result, err
	case model.SourceDesktopCache:
		return runDesktopCache(ctx, cfg, st, opts, decrypt)
	default:
		return Result{}, fmt.Errorf("source %q is disabled or unsupported in this build", opts.Source)
	}
}

func runDesktopCache(ctx context.Context, cfg config.Config, st *store.Store, opts Options, decrypt encryptedjson.DecryptFunc) (Result, error) {
	if !cfg.Granola.AllowDesktopCache {
		return Result{}, fmt.Errorf("desktop-cache source disabled in config")
	}
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	if granola.EncryptedCacheState(paths) {
		if decrypt == nil {
			return DesktopCache(ctx, cfg, st, opts)
		}
		files, err := decrypt(ctx, paths.Root, encryptedjson.CacheFile)
		if err != nil {
			return Result{Source: model.SourceEncryptedJSON}, err
		}
		defer encryptedjson.Clear(files)
		raw, ok := files[encryptedjson.CacheFile]
		if !ok {
			return Result{Source: model.SourceEncryptedJSON}, fmt.Errorf("encrypted storage decryptor omitted %s", encryptedjson.CacheFile)
		}
		return encryptedDesktopCache(ctx, st, opts, raw)
	}
	return DesktopCache(ctx, cfg, st, opts)
}

func encryptedJSONDecryptor(cfg config.Config, opts Options) (encryptedjson.DecryptFunc, error) {
	if opts.UnlockSurface == "" {
		return nil, nil
	}
	if opts.UnlockSurface != "encrypted-json" {
		return nil, fmt.Errorf("unsupported unlock surface %q", opts.UnlockSurface)
	}
	if !cfg.Granola.AllowEncryptedJSON {
		return nil, fmt.Errorf("encrypted-json source disabled in config")
	}
	if !security.PromptAllowed(cfg.Security.KeychainPromptMode) {
		return nil, fmt.Errorf("keychain prompt mode %q blocks encrypted-json unlock", cfg.Security.KeychainPromptMode)
	}
	if opts.DecryptEncryptedJSON != nil {
		return opts.DecryptEncryptedJSON, nil
	}
	return encryptedjson.Decrypt, nil
}

func privateAPIAuthUnavailable(err error) bool {
	return errors.Is(err, ErrPrivateAPITokenNotFound) || errors.Is(err, ErrPrivateAPITokenExpired)
}
