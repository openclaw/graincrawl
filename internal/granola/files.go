package granola

import (
	"fmt"
	"os"
	"strings"

	"github.com/openclaw/graincrawl/internal/model"
)

const EncryptedOnlyStateMessage = "Granola desktop state is encrypted-only; rerun with explicit encrypted-json unlock enabled to read newer cache-v6.json.enc or supabase.json.enc, or use a current plaintext source."

const PostMigrationStateMessage = "Granola 7.427+ moved its data-encryption key into the macOS data-protection Keychain under the app-scoped access group QZ7DHHLN25.granola; only Granola-signed code can read it, so graincrawl cannot decrypt local Granola state on these versions."

type FileState struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Size    int64  `json:"size,omitempty"`
	ModTime string `json:"mod_time,omitempty"`
}

func StatFile(path string) FileState {
	state := FileState{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		return state
	}
	state.Exists = true
	state.Size = info.Size()
	state.ModTime = info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	return state
}

func EncryptedNewer(plain, encrypted string) bool {
	p, perr := os.Stat(plain)
	e, eerr := os.Stat(encrypted)
	if eerr != nil {
		return false
	}
	if perr != nil {
		return true
	}
	return e.ModTime().After(p.ModTime())
}

func EncryptedOnlyState(paths ProfilePaths) bool {
	return EncryptedCacheState(paths) || EncryptedSupabaseState(paths)
}

func PostMigrationState(paths ProfilePaths) bool {
	if _, err := os.Stat(paths.StorageDEK); !os.IsNotExist(err) {
		return false
	}
	entries, err := os.ReadDir(paths.Root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".enc") {
			return true
		}
	}
	return false
}

// PlaintextSourceStateUsable reports whether the plaintext file a source reads
// actually exists and is not superseded by an encrypted copy. Absence of an
// encrypted file alone does not make a source usable.
func PlaintextSourceStateUsable(paths ProfilePaths, source model.Source) bool {
	var path string
	switch source {
	case model.SourcePrivateAPI:
		path = paths.Supabase
	case model.SourceDesktopCache:
		path = paths.CacheV6
	default:
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return !SourceStateEncrypted(paths, source)
}

// SourceStateEncrypted reports whether the local state a source needs is
// encrypted. private-api reads supabase.json, desktop-cache reads cache-v6.json;
// an encrypted file belonging to the other source must not block this one.
func SourceStateEncrypted(paths ProfilePaths, source model.Source) bool {
	switch source {
	case model.SourcePrivateAPI:
		return EncryptedSupabaseState(paths)
	case model.SourceDesktopCache:
		return EncryptedCacheState(paths)
	default:
		return EncryptedOnlyState(paths)
	}
}

func PostMigrationDiagnostic(version string) string {
	if version == "" {
		return PostMigrationStateMessage
	}
	return fmt.Sprintf("%s Detected Granola version %s.", PostMigrationStateMessage, version)
}

func EncryptedCacheState(paths ProfilePaths) bool {
	return EncryptedNewer(paths.CacheV6, paths.CacheV6Encrypted)
}

func EncryptedSupabaseState(paths ProfilePaths) bool {
	return EncryptedNewer(paths.Supabase, paths.SupabaseEncrypted)
}
