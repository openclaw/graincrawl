package syncer

import "github.com/openclaw/graincrawl/internal/model"

type Options struct {
	Source             model.Source `json:"source"`
	Limit              int          `json:"limit"`
	IncludeTranscripts bool         `json:"include_transcripts"`
	IncludePanels      bool         `json:"include_panels"`
	SkipTranscripts    bool         `json:"-"`
	SkipPanels         bool         `json:"-"`
}

type Result struct {
	Source      model.Source `json:"source"`
	Notes       int          `json:"notes"`
	Transcripts int          `json:"transcripts"`
	Panels      int          `json:"panels"`
	Message     string       `json:"message,omitempty"`
}
