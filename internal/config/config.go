package config

import (
	"fmt"
	"os"
	"path/filepath"

	ckconfig "github.com/openclaw/crawlkit/config"
	"github.com/pelletier/go-toml/v2"
)

const (
	AppName   = "graincrawl"
	ConfigEnv = "GRAINCRAWL_CONFIG"
)

type Config struct {
	Version  int            `toml:"version" json:"version"`
	Paths    RuntimePaths   `toml:"paths" json:"paths"`
	Granola  GranolaConfig  `toml:"granola" json:"granola"`
	API      APIConfig      `toml:"api" json:"api"`
	Sync     SyncConfig     `toml:"sync" json:"sync"`
	Security SecurityConfig `toml:"security" json:"security"`
}

type RuntimePaths struct {
	DBPath      string `toml:"db_path" json:"db_path"`
	CacheDir    string `toml:"cache_dir" json:"cache_dir"`
	LogDir      string `toml:"log_dir" json:"log_dir"`
	SnapshotDir string `toml:"snapshot_dir" json:"snapshot_dir"`
}

type GranolaConfig struct {
	ProfilePath        string `toml:"profile_path" json:"profile_path"`
	AppPath            string `toml:"app_path" json:"app_path"`
	PreferredSource    string `toml:"preferred_source" json:"preferred_source"`
	AllowPrivateAPI    bool   `toml:"allow_private_api" json:"allow_private_api"`
	AllowPublicAPI     bool   `toml:"allow_public_api" json:"allow_public_api"`
	AllowCompanionCLI  bool   `toml:"allow_companion_cli" json:"allow_companion_cli"`
	AllowDesktopCache  bool   `toml:"allow_desktop_cache" json:"allow_desktop_cache"`
	AllowEncryptedJSON bool   `toml:"allow_encrypted_json" json:"allow_encrypted_json"`
	AllowOPFS          bool   `toml:"allow_opfs" json:"allow_opfs"`
}

type APIConfig struct {
	ClientVersion       string `toml:"client_version" json:"client_version"`
	Platform            string `toml:"platform" json:"platform"`
	IncludeSharedWithMe bool   `toml:"include_shared_with_me" json:"include_shared_with_me"`
	RefreshMode         string `toml:"refresh_mode" json:"refresh_mode"`
}

type SyncConfig struct {
	DefaultLimit          int  `toml:"default_limit" json:"default_limit"`
	IncludeTranscripts    bool `toml:"include_transcripts" json:"include_transcripts"`
	IncludePanels         bool `toml:"include_panels" json:"include_panels"`
	IncludeCalendarEvents bool `toml:"include_calendar_events" json:"include_calendar_events"`
	IncludePeople         bool `toml:"include_people" json:"include_people"`
}

type SecurityConfig struct {
	RedactLogs         bool   `toml:"redact_logs" json:"redact_logs"`
	KeychainPromptMode string `toml:"keychain_prompt_mode" json:"keychain_prompt_mode"`
	PersistHelperKeys  bool   `toml:"persist_helper_keys" json:"persist_helper_keys"`
	DebugKeepTemp      bool   `toml:"debug_keep_temp" json:"debug_keep_temp"`
}

func App() ckconfig.App {
	return ckconfig.App{Name: AppName, ConfigEnv: ConfigEnv}
}

func Defaults() (Config, string, error) {
	paths, err := App().DefaultPaths()
	if err != nil {
		return Config{}, "", err
	}
	cfg := Config{
		Version: 1,
		Paths: RuntimePaths{
			DBPath:      envOr("GRAINCRAWL_DB_PATH", paths.DBPath),
			CacheDir:    paths.CacheDir,
			LogDir:      paths.LogDir,
			SnapshotDir: filepath.Join(paths.BaseDir, "snapshots"),
		},
		Granola: GranolaConfig{
			ProfilePath:       envOr("GRAINCRAWL_GRANOLA_PROFILE", defaultGranolaProfile()),
			AppPath:           "/Applications/Granola.app",
			PreferredSource:   envOr("GRAINCRAWL_SOURCE", "private-api"),
			AllowPrivateAPI:   envBool("GRAINCRAWL_ALLOW_PRIVATE_API", true),
			AllowCompanionCLI: true,
			AllowDesktopCache: true,
		},
		API: APIConfig{
			ClientVersion:       "auto",
			Platform:            "darwin",
			IncludeSharedWithMe: true,
			RefreshMode:         "never",
		},
		Sync: SyncConfig{
			DefaultLimit:          100,
			IncludeTranscripts:    true,
			IncludePanels:         true,
			IncludeCalendarEvents: true,
			IncludePeople:         true,
		},
		Security: SecurityConfig{
			RedactLogs:         true,
			KeychainPromptMode: "explicit",
		},
	}
	return cfg, paths.ConfigPath, nil
}

func Load(path string) (Config, string, error) {
	cfg, defaultPath, err := Defaults()
	if err != nil {
		return Config{}, "", err
	}
	resolved, err := App().ResolveConfigPath(path)
	if err != nil {
		return Config{}, "", err
	}
	if resolved == "" {
		resolved = defaultPath
	}
	if b, err := os.ReadFile(resolved); err == nil {
		if err := toml.Unmarshal(b, &cfg); err != nil {
			return Config{}, resolved, fmt.Errorf("parse config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, resolved, err
	}
	return cfg, resolved, nil
}

func Save(path string, cfg Config) error {
	b, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func EnsureDirs(cfg Config) error {
	for _, dir := range []string{filepath.Dir(cfg.Paths.DBPath), cfg.Paths.CacheDir, cfg.Paths.LogDir, cfg.Paths.SnapshotDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	switch os.Getenv(name) {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	case "0", "false", "FALSE", "no", "NO":
		return false
	default:
		return fallback
	}
}

func defaultGranolaProfile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "Granola")
}
