package cachev6

import (
	"encoding/json"
	"fmt"
	"os"
)

func Read(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(b, &file); err != nil {
		return File{}, err
	}
	if file.Cache.Version != 6 {
		return File{}, fmt.Errorf("unsupported cache version %d", file.Cache.Version)
	}
	if file.Cache.State.Documents == nil {
		file.Cache.State.Documents = map[string]Document{}
	}
	if file.Cache.State.Transcripts == nil {
		file.Cache.State.Transcripts = map[string][]TranscriptChunk{}
	}
	return file, nil
}

type Summary struct {
	Version          int  `json:"version"`
	DocumentCount    int  `json:"document_count"`
	TranscriptDocs   int  `json:"transcript_docs"`
	MeetingMetaCount int  `json:"meeting_meta_count"`
	EncryptedCache   bool `json:"encrypted_cache_storage"`
	EncryptedAuth    bool `json:"encrypted_supabase_storage"`
}

func Summarize(file File) Summary {
	return Summary{
		Version:          file.Cache.Version,
		DocumentCount:    len(file.Cache.State.Documents),
		TranscriptDocs:   len(file.Cache.State.Transcripts),
		MeetingMetaCount: len(file.Cache.State.MeetingsMetadata),
		EncryptedCache:   truthy(file.Cache.State.FeatureFlags["encrypted_cache_storage"]),
		EncryptedAuth:    truthy(file.Cache.State.FeatureFlags["encrypted_supabase_storage"]),
	}
}

func truthy(value any) bool {
	v, ok := value.(bool)
	return ok && v
}
