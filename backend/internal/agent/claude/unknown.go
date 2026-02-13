// Package claudecode provides Go types for Claude Code JSONL session logs.
//
// Claude Code writes JSONL files that record conversation sessions including
// user messages, assistant responses, tool usage, progress events, and more.
// New fields may appear at any version, so all types preserve unknown fields
// in an Overflow map and log a warning when they are encountered.
package claude

import (
	"encoding/json"
	"log/slog"
	"sort"
)

// Overflow holds JSON fields that were not mapped to a struct field.
// It is embedded in every record type to ensure forward compatibility.
type Overflow struct {
	// Extra contains any JSON fields not recognized by the current struct definition.
	// These are preserved during unmarshaling so no data is lost.
	Extra map[string]json.RawMessage `json:"-"`
}

// warnUnknown logs a warning for each key in extra, identified by context.
func warnUnknown(context string, extra map[string]json.RawMessage) {
	if len(extra) == 0 {
		return
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	slog.Warn("unknown fields in Claude Code record", "context", context, "fields", keys)
}
