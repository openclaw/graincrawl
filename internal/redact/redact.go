package redact

import "strings"

const Mask = "[redacted]"

var sensitiveKeys = []string{
	"access_token",
	"refresh_token",
	"token",
	"authorization",
	"email",
	"notes",
	"notes_plain",
	"notes_markdown",
	"transcript",
	"text",
	"content",
}

func Key(key string) bool {
	key = strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if key == sensitive || strings.Contains(key, sensitive) {
			return true
		}
	}
	return false
}
