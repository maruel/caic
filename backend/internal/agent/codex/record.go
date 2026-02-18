package codex

import (
	"encoding/json"
	"fmt"
)

// Event type constants for the outer envelope.
const (
	TypeThreadStarted = "thread.started"
	TypeTurnStarted   = "turn.started"
	TypeTurnCompleted = "turn.completed"
	TypeTurnFailed    = "turn.failed"
	TypeItemStarted   = "item.started"
	TypeItemUpdated   = "item.updated"
	TypeItemCompleted = "item.completed"
	TypeError         = "error"
)

// Item type constants for the inner item object.
const (
	ItemTypeAgentMessage     = "agent_message"
	ItemTypeReasoning        = "reasoning"
	ItemTypeCommandExecution = "command_execution"
	ItemTypeFileChange       = "file_change"
	ItemTypeMCPToolCall      = "mcp_tool_call"
	ItemTypeWebSearch        = "web_search"
	ItemTypeTodoList         = "todo_list"
	ItemTypeError            = "error"
)

// Record is a single line from a Codex CLI exec --json session.
// Use the typed accessor methods to get the concrete record after checking Type.
type Record struct {
	// Type discriminates the record kind.
	Type string `json:"type"`

	raw json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Record) UnmarshalJSON(data []byte) error {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("Record: %w", err)
	}
	r.Type = probe.Type
	r.raw = append(r.raw[:0], data...)
	return nil
}

// Raw returns the original JSON bytes for this record.
func (r *Record) Raw() json.RawMessage { return r.raw }

// AsThreadStarted decodes the record as a ThreadStartedRecord.
func (r *Record) AsThreadStarted() (*ThreadStartedRecord, error) {
	var v ThreadStartedRecord
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// AsTurnCompleted decodes the record as a TurnCompletedRecord.
func (r *Record) AsTurnCompleted() (*TurnCompletedRecord, error) {
	var v TurnCompletedRecord
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// AsTurnFailed decodes the record as a TurnFailedRecord.
func (r *Record) AsTurnFailed() (*TurnFailedRecord, error) {
	var v TurnFailedRecord
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// AsItem decodes the record as an ItemRecord.
func (r *Record) AsItem() (*ItemRecord, error) {
	var v ItemRecord
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ThreadStartedRecord is emitted at session start.
//
// Example:
//
//	{"type":"thread.started","thread_id":"0199a213-81c0-7800-8aa1-bbab2a035a53"}
type ThreadStartedRecord struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`

	Overflow
}

var threadStartedKnown = makeSet("type", "thread_id")

// UnmarshalJSON implements json.Unmarshaler.
func (r *ThreadStartedRecord) UnmarshalJSON(data []byte) error {
	type Alias ThreadStartedRecord
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ThreadStartedRecord: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return fmt.Errorf("ThreadStartedRecord: %w", err)
	}
	r.Extra = collectUnknown(raw, threadStartedKnown)
	warnUnknown("ThreadStartedRecord", r.Extra)
	return nil
}

// TurnCompletedRecord is emitted when a turn ends successfully.
//
// Example:
//
//	{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122}}
type TurnCompletedRecord struct {
	Type  string    `json:"type"`
	Usage TurnUsage `json:"usage"`

	Overflow
}

var turnCompletedKnown = makeSet("type", "usage")

// UnmarshalJSON implements json.Unmarshaler.
func (r *TurnCompletedRecord) UnmarshalJSON(data []byte) error {
	type Alias TurnCompletedRecord
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnCompletedRecord: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return fmt.Errorf("TurnCompletedRecord: %w", err)
	}
	r.Extra = collectUnknown(raw, turnCompletedKnown)
	warnUnknown("TurnCompletedRecord", r.Extra)
	return nil
}

// TurnUsage contains token counts for a single turn.
type TurnUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`

	Overflow
}

var turnUsageKnown = makeSet("input_tokens", "cached_input_tokens", "output_tokens")

// UnmarshalJSON implements json.Unmarshaler.
func (u *TurnUsage) UnmarshalJSON(data []byte) error {
	type Alias TurnUsage
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnUsage: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(u)); err != nil {
		return fmt.Errorf("TurnUsage: %w", err)
	}
	u.Extra = collectUnknown(raw, turnUsageKnown)
	warnUnknown("TurnUsage", u.Extra)
	return nil
}

// TurnFailedRecord is emitted when a turn errors.
//
// Example:
//
//	{"type":"turn.failed","error":"something went wrong"}
type TurnFailedRecord struct {
	Type  string `json:"type"`
	Error string `json:"error"`

	Overflow
}

var turnFailedKnown = makeSet("type", "error")

// UnmarshalJSON implements json.Unmarshaler.
func (r *TurnFailedRecord) UnmarshalJSON(data []byte) error {
	type Alias TurnFailedRecord
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnFailedRecord: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return fmt.Errorf("TurnFailedRecord: %w", err)
	}
	r.Extra = collectUnknown(raw, turnFailedKnown)
	warnUnknown("TurnFailedRecord", r.Extra)
	return nil
}

// ItemRecord is emitted for item.started, item.updated, and item.completed
// events. The Item field contains the inner item data.
//
// Example:
//
//	{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"docs\nsrc\n","exit_code":0,"status":"completed"}}
type ItemRecord struct {
	Type string   `json:"type"`
	Item ItemData `json:"item"`

	Overflow
}

var itemRecordKnown = makeSet("type", "item")

// UnmarshalJSON implements json.Unmarshaler.
func (r *ItemRecord) UnmarshalJSON(data []byte) error {
	type Alias ItemRecord
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ItemRecord: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return fmt.Errorf("ItemRecord: %w", err)
	}
	r.Extra = collectUnknown(raw, itemRecordKnown)
	warnUnknown("ItemRecord", r.Extra)
	return nil
}

// ItemData is the inner item object within item events.
type ItemData struct {
	ID     string `json:"id"`
	Type   string `json:"type"`   // agent_message, reasoning, command_execution, file_change, mcp_tool_call, web_search, todo_list, error
	Status string `json:"status"` // in_progress, completed, failed

	// agent_message / reasoning fields.
	Text string `json:"text,omitempty"`

	// command_execution fields.
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`

	// file_change fields.
	Changes []FileChange `json:"changes,omitempty"`

	// mcp_tool_call fields.
	Server    string          `json:"server,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    string          `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`

	// web_search fields.
	Query string `json:"query,omitempty"`

	// todo_list fields.
	Items []TodoItem `json:"items,omitempty"`

	// error fields (when type=error).
	Message string `json:"message,omitempty"`

	Overflow
}

var itemDataKnown = makeSet(
	"id", "type", "status", "text",
	"command", "aggregated_output", "exit_code",
	"changes",
	"server", "tool", "arguments", "result", "error",
	"query",
	"items",
	"message",
)

// UnmarshalJSON implements json.Unmarshaler.
func (d *ItemData) UnmarshalJSON(data []byte) error {
	type Alias ItemData
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ItemData: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(d)); err != nil {
		return fmt.Errorf("ItemData: %w", err)
	}
	d.Extra = collectUnknown(raw, itemDataKnown)
	warnUnknown("ItemData("+d.Type+")", d.Extra)
	return nil
}

// FileChange describes a single file change within a file_change item.
type FileChange struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // add, update, delete
}

// TodoItem is a single entry in a todo_list item.
type TodoItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}
