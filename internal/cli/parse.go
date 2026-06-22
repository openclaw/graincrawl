package cli

import (
	"fmt"
	"strconv"

	"github.com/openclaw/graincrawl/internal/model"
	"github.com/openclaw/graincrawl/internal/syncer"
)

func parseSyncOptions(args []string) (syncer.Options, error) {
	opts := syncer.Options{}
	if source, ok := flagValue(args, "--source"); ok {
		opts.Source = model.Source(source)
	}
	if limit, ok := flagValue(args, "--limit"); ok {
		n, err := strconv.Atoi(limit)
		if err != nil || n <= 0 {
			return opts, fmt.Errorf("--limit must be a positive integer")
		}
		opts.Limit = n
	}
	if surface, ok := flagValue(args, "--unlock"); ok {
		opts.UnlockSurface = surface
	}
	opts.SkipTranscripts = hasFlag(args, "--no-transcripts")
	opts.SkipPanels = hasFlag(args, "--no-panels")
	return opts, nil
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
