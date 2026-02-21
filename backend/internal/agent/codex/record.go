package codex

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC notification method constants for codex app-server.
const (
	MethodThreadStarted     = "thread/started"
	MethodTurnStarted       = "turn/started"
	MethodTurnCompleted     = "turn/completed"
	MethodItemStarted       = "item/started"
	MethodItemCompleted     = "item/completed"
	MethodItemUpdated       = "item/updated"
	MethodItemDelta         = "item/agentMessage/delta"
	MethodTokenUsageUpdated = "thread/tokenUsage/updated"
)

// Item type constants for ThreadItem.Type (camelCase as emitted by Codex v2).
const (
	ItemTypeUserMessage       = "userMessage"
	ItemTypeAgentMessage      = "agentMessage"
	ItemTypePlan              = "plan"
	ItemTypeReasoning         = "reasoning"
	ItemTypeCommandExecution  = "commandExecution"
	ItemTypeFileChange        = "fileChange"
	ItemTypeMCPToolCall       = "mcpToolCall"
	ItemTypeWebSearch         = "webSearch"
	ItemTypeImageView         = "imageView"
	ItemTypeContextCompaction = "contextCompaction"
)

// JSONRPCMessage is the JSON-RPC 2.0 envelope for codex app-server messages.
// Notifications have Method set and ID nil. Responses have ID set.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method,omitempty"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// IsResponse returns true if this is a response (has an ID).
func (m *JSONRPCMessage) IsResponse() bool { return m.ID != nil }

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ThreadStartedParams holds the params for thread/started notifications.
type ThreadStartedParams struct {
	Thread ThreadInfo `json:"thread"`

	Overflow
}

var threadStartedParamsKnown = makeSet("thread")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ThreadStartedParams) UnmarshalJSON(data []byte) error {
	type Alias ThreadStartedParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ThreadStartedParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("ThreadStartedParams: %w", err)
	}
	p.Extra = collectUnknown(raw, threadStartedParamsKnown)
	warnUnknown("ThreadStartedParams", p.Extra)
	return nil
}

// ThreadInfo describes a thread in thread/started params.
type ThreadInfo struct {
	ID            string          `json:"id"`
	CLIVersion    string          `json:"cliVersion,omitempty"`
	CreatedAt     int64           `json:"createdAt,omitempty"` // Unix timestamp seconds.
	CWD           string          `json:"cwd,omitempty"`
	GitInfo       json.RawMessage `json:"gitInfo,omitempty"`
	ModelProvider string          `json:"modelProvider,omitempty"`
	Path          string          `json:"path,omitempty"`
	Preview       string          `json:"preview,omitempty"`
	Source        string          `json:"source,omitempty"`
	UpdatedAt     int64           `json:"updatedAt,omitempty"` // Unix timestamp seconds.

	Overflow
}

var threadInfoKnown = makeSet(
	"id", "cliVersion", "createdAt", "cwd", "gitInfo", "modelProvider",
	"path", "preview", "source", "updatedAt",
	// v2 additional fields not captured above.
	"status", "name", "agentNickname", "agentRole", "turns",
)

// UnmarshalJSON implements json.Unmarshaler.
func (t *ThreadInfo) UnmarshalJSON(data []byte) error {
	type Alias ThreadInfo
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ThreadInfo: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(t)); err != nil {
		return fmt.Errorf("ThreadInfo: %w", err)
	}
	t.Extra = collectUnknown(raw, threadInfoKnown)
	warnUnknown("ThreadInfo", t.Extra)
	return nil
}

// TurnStartedParams holds the params for turn/started notifications.
type TurnStartedParams struct {
	ThreadID string   `json:"threadId"`
	Turn     TurnInfo `json:"turn"`

	Overflow
}

var turnStartedParamsKnown = makeSet("threadId", "turn")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TurnStartedParams) UnmarshalJSON(data []byte) error {
	type Alias TurnStartedParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnStartedParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("TurnStartedParams: %w", err)
	}
	p.Extra = collectUnknown(raw, turnStartedParamsKnown)
	warnUnknown("TurnStartedParams", p.Extra)
	return nil
}

// TurnCompletedParams holds the params for turn/completed notifications.
type TurnCompletedParams struct {
	ThreadID string   `json:"threadId"`
	Turn     TurnInfo `json:"turn"`

	Overflow
}

var turnCompletedParamsKnown = makeSet("threadId", "turn")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TurnCompletedParams) UnmarshalJSON(data []byte) error {
	type Alias TurnCompletedParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnCompletedParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("TurnCompletedParams: %w", err)
	}
	p.Extra = collectUnknown(raw, turnCompletedParamsKnown)
	warnUnknown("TurnCompletedParams", p.Extra)
	return nil
}

// TurnInfo describes a turn in turn/started and turn/completed params.
type TurnInfo struct {
	ID     string     `json:"id"`
	Status string     `json:"status"` // "completed", "failed", "interrupted", "inProgress"
	Error  *TurnError `json:"error,omitempty"`

	Overflow
}

var turnInfoKnown = makeSet("id", "status", "error", "items")

// UnmarshalJSON implements json.Unmarshaler.
func (t *TurnInfo) UnmarshalJSON(data []byte) error {
	type Alias TurnInfo
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnInfo: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(t)); err != nil {
		return fmt.Errorf("TurnInfo: %w", err)
	}
	t.Extra = collectUnknown(raw, turnInfoKnown)
	warnUnknown("TurnInfo", t.Extra)
	return nil
}

// TurnError describes a turn failure.
type TurnError struct {
	Message           string `json:"message"`
	AdditionalDetails string `json:"additionalDetails,omitempty"`

	Overflow
}

var turnErrorKnown = makeSet("message", "codexErrorInfo", "additionalDetails")

// UnmarshalJSON implements json.Unmarshaler.
func (e *TurnError) UnmarshalJSON(data []byte) error {
	type Alias TurnError
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TurnError: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(e)); err != nil {
		return fmt.Errorf("TurnError: %w", err)
	}
	e.Extra = collectUnknown(raw, turnErrorKnown)
	warnUnknown("TurnError", e.Extra)
	return nil
}

// ItemParams holds the params for item/started and item/completed notifications.
type ItemParams struct {
	Item     ThreadItem `json:"item"`
	ThreadID string     `json:"threadId"`
	TurnID   string     `json:"turnId"`

	Overflow
}

var itemParamsKnown = makeSet("item", "threadId", "turnId")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ItemParams) UnmarshalJSON(data []byte) error {
	type Alias ItemParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ItemParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("ItemParams: %w", err)
	}
	p.Extra = collectUnknown(raw, itemParamsKnown)
	warnUnknown("ItemParams", p.Extra)
	return nil
}

// ItemDeltaParams holds the params for item/agentMessage/delta notifications.
type ItemDeltaParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`

	Overflow
}

var itemDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ItemDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias ItemDeltaParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ItemDeltaParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("ItemDeltaParams: %w", err)
	}
	p.Extra = collectUnknown(raw, itemDeltaParamsKnown)
	warnUnknown("ItemDeltaParams", p.Extra)
	return nil
}

// ThreadItem is the discriminated-union item type used in item/* notifications.
// The Type field is the discriminant (camelCase, e.g. "agentMessage").
type ThreadItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	// agentMessage / plan fields.
	Text string `json:"text,omitempty"`

	// reasoning fields.
	Summary []string        `json:"summary,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`

	// commandExecution fields.
	Command          string  `json:"command,omitempty"`
	AggregatedOutput *string `json:"aggregatedOutput,omitempty"` // nullable
	ExitCode         *int    `json:"exitCode,omitempty"`

	// fileChange fields.
	Changes []FileUpdateChange `json:"changes,omitempty"`

	// mcpToolCall fields.
	Server    string             `json:"server,omitempty"`
	Tool      string             `json:"tool,omitempty"`
	Arguments json.RawMessage    `json:"arguments,omitempty"`
	Result    *McpToolCallResult `json:"result,omitempty"`
	Error     *McpToolCallError  `json:"error,omitempty"`

	// webSearch fields.
	Query string `json:"query,omitempty"`

	Overflow
}

var threadItemKnown = makeSet(
	"id", "type",
	"text", "phase", // agentMessage / plan
	"summary", "content", // reasoning
	"command", "cwd", "processId", "status", "commandActions", // commandExecution
	"aggregatedOutput", "exitCode", "durationMs",
	"changes",                                        // fileChange
	"server", "tool", "arguments", "result", "error", // mcpToolCall
	"query", "action", // webSearch
	"path",                                                          // imageView
	"review",                                                        // enteredReviewMode / exitedReviewMode
	"senderThreadId", "receiverThreadIds", "prompt", "agentsStates", // collabAgentToolCall
)

// UnmarshalJSON implements json.Unmarshaler.
func (d *ThreadItem) UnmarshalJSON(data []byte) error {
	type Alias ThreadItem
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ThreadItem: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(d)); err != nil {
		return fmt.Errorf("ThreadItem: %w", err)
	}
	d.Extra = collectUnknown(raw, threadItemKnown)
	warnUnknown("ThreadItem("+d.Type+")", d.Extra)
	return nil
}

// FileUpdateChange describes a single file change within a fileChange item.
type FileUpdateChange struct {
	Path string          `json:"path"`
	Kind PatchChangeKind `json:"kind"`
	Diff string          `json:"diff,omitempty"`
}

// PatchChangeKind is the discriminated kind for FileUpdateChange.
type PatchChangeKind struct {
	Type     string  `json:"type"`               // "add", "delete", "update"
	MovePath *string `json:"movePath,omitempty"` // only for "update" with rename
}

// McpToolCallResult holds the result of a successful MCP tool call.
type McpToolCallResult struct {
	Content           []json.RawMessage `json:"content"`
	StructuredContent json.RawMessage   `json:"structuredContent,omitempty"`
}

// McpToolCallError holds the error from a failed MCP tool call.
type McpToolCallError struct {
	Message string `json:"message"`
}

// TokenUsageUpdatedParams holds params for thread/tokenUsage/updated notifications.
type TokenUsageUpdatedParams struct {
	ThreadID   string           `json:"threadId"`
	TurnID     string           `json:"turnId"`
	TokenUsage ThreadTokenUsage `json:"tokenUsage"`

	Overflow
}

var tokenUsageUpdatedParamsKnown = makeSet("threadId", "turnId", "tokenUsage")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TokenUsageUpdatedParams) UnmarshalJSON(data []byte) error {
	type Alias TokenUsageUpdatedParams
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("TokenUsageUpdatedParams: %w", err)
	}
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return fmt.Errorf("TokenUsageUpdatedParams: %w", err)
	}
	p.Extra = collectUnknown(raw, tokenUsageUpdatedParamsKnown)
	warnUnknown("TokenUsageUpdatedParams", p.Extra)
	return nil
}

// ThreadTokenUsage holds cumulative and per-turn token usage for a thread.
type ThreadTokenUsage struct {
	Total              TokenUsageBreakdown `json:"total"`
	Last               TokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64              `json:"modelContextWindow,omitempty"`
}

// TokenUsageBreakdown contains a detailed breakdown of token counts.
type TokenUsageBreakdown struct {
	TotalTokens           int64 `json:"totalTokens"`
	InputTokens           int64 `json:"inputTokens"`
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
}
