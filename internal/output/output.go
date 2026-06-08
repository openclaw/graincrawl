package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Mode string

const (
	ModeText Mode = "text"
	ModeJSON Mode = "json"
)

type Envelope struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func WriteEnvelope(w io.Writer, value any) error {
	return WriteJSON(w, Envelope{OK: true, Result: value})
}

func PrintKV(w io.Writer, key string, value any) {
	fmt.Fprintf(w, "%s: %v\n", key, value)
}
