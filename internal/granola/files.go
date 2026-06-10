package granola

import "os"

const EncryptedOnlyStateMessage = "Granola desktop state is encrypted-only; encrypted-json unlock/import is not implemented in this version; re-signing into Granola desktop will not fix this unless it emits plaintext supabase.json/cache-v6.json."

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

func EncryptedCacheState(paths ProfilePaths) bool {
	return EncryptedNewer(paths.CacheV6, paths.CacheV6Encrypted)
}

func EncryptedSupabaseState(paths ProfilePaths) bool {
	return EncryptedNewer(paths.Supabase, paths.SupabaseEncrypted)
}
