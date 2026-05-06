package granola

import "os"

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
