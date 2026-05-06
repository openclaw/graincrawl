package granola

import (
	"os"
	"path/filepath"
)

const (
	DefaultAppPath = "/Applications/Granola.app"
	BundleID       = "com.granola.app"
)

type ProfilePaths struct {
	Root                string `json:"root"`
	CacheV6             string `json:"cache_v6"`
	CacheV6Encrypted    string `json:"cache_v6_encrypted"`
	Supabase            string `json:"supabase"`
	SupabaseEncrypted   string `json:"supabase_encrypted"`
	StorageDEK          string `json:"storage_dek"`
	UserPreferences     string `json:"user_preferences"`
	UserPreferencesEnc  string `json:"user_preferences_encrypted"`
	IndexedDB           string `json:"indexed_db"`
	FileSystem          string `json:"file_system"`
	CompanionMetadata   string `json:"companion_metadata"`
	CompanionBinaryPath string `json:"companion_binary_path"`
}

func DefaultProfilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "Granola")
}

func Paths(profile, appPath string) ProfilePaths {
	if profile == "" {
		profile = DefaultProfilePath()
	}
	if appPath == "" {
		appPath = DefaultAppPath
	}
	return ProfilePaths{
		Root:                profile,
		CacheV6:             filepath.Join(profile, "cache-v6.json"),
		CacheV6Encrypted:    filepath.Join(profile, "cache-v6.json.enc"),
		Supabase:            filepath.Join(profile, "supabase.json"),
		SupabaseEncrypted:   filepath.Join(profile, "supabase.json.enc"),
		StorageDEK:          filepath.Join(profile, "storage.dek"),
		UserPreferences:     filepath.Join(profile, "user-preferences.json"),
		UserPreferencesEnc:  filepath.Join(profile, "user-preferences.json.enc"),
		IndexedDB:           filepath.Join(profile, "IndexedDB"),
		FileSystem:          filepath.Join(profile, "File System"),
		CompanionMetadata:   filepath.Join(profile, "companion-cli", "companion-cli.json"),
		CompanionBinaryPath: filepath.Join(appPath, "Contents", "Resources", "bin", "granola"),
	}
}
