package cli

import (
	"strconv"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/syncer"
)

func parseSyncOptions(args []string) syncer.Options {
	opts := syncer.Options{}
	if source, ok := flagValue(args, "--source"); ok {
		opts.Source = model.Source(source)
	}
	if limit, ok := flagValue(args, "--limit"); ok {
		if n, err := strconv.Atoi(limit); err == nil {
			opts.Limit = n
		}
	}
	opts.IncludeTranscripts = !hasFlag(args, "--no-transcripts")
	opts.IncludePanels = !hasFlag(args, "--no-panels")
	return opts
}

func parseLimit(args []string, fallback int) int {
	if limit, ok := flagValue(args, "--limit"); ok {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func parseOutDir(args []string) string {
	if out, ok := flagValue(args, "--out"); ok {
		return out
	}
	return ""
}
