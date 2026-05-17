package doctor

import (
	"os"
	"time"

	"github.com/openclaw/graincrawl/internal/config"
	"github.com/openclaw/graincrawl/internal/granola"
)

type Report struct {
	ConfigPath string                `json:"config_path"`
	DBPath     string                `json:"db_path"`
	GranolaApp granola.AppInfo       `json:"granola_app"`
	Profile    granola.ProfilePaths  `json:"profile"`
	Files      FileReport            `json:"files"`
	Token      *granola.TokenSummary `json:"token,omitempty"`
	Unlock     UnlockReport          `json:"unlock"`
}

type FileReport struct {
	CacheV6           granola.FileState `json:"cache_v6"`
	CacheV6Encrypted  granola.FileState `json:"cache_v6_encrypted"`
	Supabase          granola.FileState `json:"supabase"`
	SupabaseEncrypted granola.FileState `json:"supabase_encrypted"`
	StorageDEK        granola.FileState `json:"storage_dek"`
	IndexedDB         granola.FileState `json:"indexed_db"`
	FileSystem        granola.FileState `json:"file_system"`
	CompanionBinary   granola.FileState `json:"companion_binary"`
}

type UnlockReport struct {
	EncryptedJSONRequired bool `json:"encrypted_json_required"`
	OPFSPresent           bool `json:"opfs_present"`
	KeychainMayPrompt     bool `json:"keychain_may_prompt"`
}

func Run(cfg config.Config, configPath string, now time.Time) Report {
	paths := granola.Paths(cfg.Granola.ProfilePath, cfg.Granola.AppPath)
	report := Report{
		ConfigPath: configPath,
		DBPath:     cfg.Paths.DBPath,
		GranolaApp: granola.InspectApp(cfg.Granola.AppPath),
		Profile:    paths,
		Files: FileReport{
			CacheV6:           granola.StatFile(paths.CacheV6),
			CacheV6Encrypted:  granola.StatFile(paths.CacheV6Encrypted),
			Supabase:          granola.StatFile(paths.Supabase),
			SupabaseEncrypted: granola.StatFile(paths.SupabaseEncrypted),
			StorageDEK:        granola.StatFile(paths.StorageDEK),
			IndexedDB:         statDir(paths.IndexedDB),
			FileSystem:        statDir(paths.FileSystem),
			CompanionBinary:   granola.StatFile(paths.CompanionBinaryPath),
		},
	}
	if _, tokens, _, err := granola.ReadSupabase(paths.Supabase); err == nil {
		summary := granola.SummarizeToken(tokens, now)
		report.Token = &summary
	}
	report.Unlock.EncryptedJSONRequired = granola.EncryptedNewer(paths.CacheV6, paths.CacheV6Encrypted) || granola.EncryptedNewer(paths.Supabase, paths.SupabaseEncrypted)
	report.Unlock.OPFSPresent = report.Files.IndexedDB.Exists && report.Files.FileSystem.Exists
	report.Unlock.KeychainMayPrompt = report.Unlock.EncryptedJSONRequired || cfg.Granola.AllowEncryptedJSON || cfg.Granola.AllowOPFS
	return report
}

func statDir(path string) granola.FileState {
	state := granola.StatFile(path)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		state.Exists = true
	}
	return state
}
