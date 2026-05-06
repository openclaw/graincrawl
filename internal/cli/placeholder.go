package cli

import (
	"io"

	"github.com/vincentkoc/graincrawl/internal/output"
)

type placeholderResult struct {
	Command     string `json:"command"`
	Ready       bool   `json:"ready"`
	Description string `json:"description"`
}

func (a App) runPlaceholder(w io.Writer, flags GlobalFlags, command, description string) error {
	result := placeholderResult{
		Command:     command,
		Ready:       false,
		Description: description,
	}
	if flags.JSON {
		return output.WriteEnvelope(w, result)
	}
	output.PrintKV(w, "command", command)
	output.PrintKV(w, "ready", false)
	output.PrintKV(w, "status", description)
	return nil
}
