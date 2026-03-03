// Wire types for the Codex CLI app-server JSON-RPC 2.0 protocol.
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

	MethodCommandOutputDelta        = "item/commandExecution/outputDelta"
	MethodCommandTerminalInteract   = "item/commandExecution/terminalInteraction"
	MethodFileChangeOutputDelta     = "item/fileChange/outputDelta"
	MethodReasoningSummaryTextDelta = "item/reasoning/summaryTextDelta"
	MethodReasoningSummaryPartAdded = "item/reasoning/summaryPartAdded"
	MethodReasoningTextDelta        = "item/reasoning/textDelta"
	MethodPlanDelta                 = "item/plan/delta"
	MethodMcpToolCallProgress       = "item/mcpToolCall/progress"
	MethodTurnDiffUpdated           = "turn/diff/updated"
	MethodTurnPlanUpdated           = "turn/plan/updated"
	MethodThreadStatusChanged       = "thread/status/changed"
	MethodThreadNameUpdated         = "thread/name/updated"
	MethodModelRerouted             = "model/rerouted"
	MethodErrorNotification         = "error"
)

// Item type constants (camelCase as emitted by Codex v2).
const (
	ItemTypeUserMessage         = "userMessage"
	ItemTypeAgentMessage        = "agentMessage"
	ItemTypePlan                = "plan"
	ItemTypeReasoning           = "reasoning"
	ItemTypeCommandExecution    = "commandExecution"
	ItemTypeFileChange          = "fileChange"
	ItemTypeMCPToolCall         = "mcpToolCall"
	ItemTypeWebSearch           = "webSearch"
	ItemTypeImageView           = "imageView"
	ItemTypeContextCompaction   = "contextCompaction"
	ItemTypeDynamicToolCall     = "dynamicToolCall"
	ItemTypeCollabAgentToolCall = "collabAgentToolCall"
	ItemTypeEnteredReviewMode   = "enteredReviewMode"
	ItemTypeExitedReviewMode    = "exitedReviewMode"
)

// unmarshalRecord decodes data into dest (which must be a type-alias pointer
// to break recursive UnmarshalJSON), collects unknown fields into overflow,
// and logs a warning for each unknown key.
func unmarshalRecord(data []byte, dest any, overflow *Overflow, known map[string]struct{}, name string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	overflow.Extra = collectUnknown(raw, known)
	warnUnknown(name, overflow.Extra)
	return nil
}

// ---------- JSON-RPC envelope ----------

// JSONRPCMessage is the JSON-RPC 2.0 envelope for codex app-server messages.
// Notifications have Method set and ID nil. Responses have ID set.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method,omitzero"`
	ID      *json.RawMessage `json:"id,omitzero"`
	Params  json.RawMessage  `json:"params,omitzero"`
	Result  json.RawMessage  `json:"result,omitzero"`
	Error   *JSONRPCError    `json:"error,omitzero"`
}

// IsResponse returns true if this is a response (has an ID).
func (m *JSONRPCMessage) IsResponse() bool { return m.ID != nil }

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------- Routing probes ----------

// messageProbe extracts routing fields from a codex app-server line to
// distinguish caic-injected JSON (has "type") from JSON-RPC (has "method"/"id").
type messageProbe struct {
	Type   string           `json:"type,omitzero"`
	Method string           `json:"method,omitzero"`
	ID     *json.RawMessage `json:"id,omitzero"`
}

// methodProbe extracts the method field from a JSON-RPC message.
type methodProbe struct {
	Method string `json:"method,omitzero"`
}

// ---------- Handshake types ----------

// threadStartResult is the result object from a thread/start JSON-RPC response.
type threadStartResult struct {
	Thread threadStartThread `json:"thread"`
}

// threadStartThread is the thread object inside a threadStartResult.
type threadStartThread struct {
	ID string `json:"id"`
}

// ---------- Thread lifecycle ----------

// ThreadStartedParams holds the params for thread/started notifications.
type ThreadStartedParams struct {
	Thread ThreadInfo `json:"thread"`
	Overflow
}

var threadStartedParamsKnown = makeSet("thread")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ThreadStartedParams) UnmarshalJSON(data []byte) error {
	type Alias ThreadStartedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, threadStartedParamsKnown, "ThreadStartedParams")
}

// ThreadInfo describes a thread in thread/started params.
type ThreadInfo struct {
	ID            string          `json:"id"`
	CLIVersion    string          `json:"cliVersion,omitzero"`
	CreatedAt     int64           `json:"createdAt,omitzero"`
	CWD           string          `json:"cwd,omitzero"`
	GitInfo       json.RawMessage `json:"gitInfo,omitzero"`
	ModelProvider string          `json:"modelProvider,omitzero"`
	Path          string          `json:"path,omitzero"`
	Preview       string          `json:"preview,omitzero"`
	Source        string          `json:"source,omitzero"`
	UpdatedAt     int64           `json:"updatedAt,omitzero"`
	Overflow
}

var threadInfoKnown = makeSet(
	"id", "cliVersion", "createdAt", "cwd", "gitInfo", "modelProvider",
	"path", "preview", "source", "updatedAt",
	"status", "name", "agentNickname", "agentRole", "turns",
)

// UnmarshalJSON implements json.Unmarshaler.
func (t *ThreadInfo) UnmarshalJSON(data []byte) error {
	type Alias ThreadInfo
	return unmarshalRecord(data, (*Alias)(t), &t.Overflow, threadInfoKnown, "ThreadInfo")
}

// ---------- Turn lifecycle ----------

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
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, turnStartedParamsKnown, "TurnStartedParams")
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
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, turnCompletedParamsKnown, "TurnCompletedParams")
}

// TurnInfo describes a turn in turn/started and turn/completed params.
type TurnInfo struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Error  *TurnError `json:"error,omitzero"`
	Overflow
}

var turnInfoKnown = makeSet("id", "status", "error", "items")

// UnmarshalJSON implements json.Unmarshaler.
func (t *TurnInfo) UnmarshalJSON(data []byte) error {
	type Alias TurnInfo
	return unmarshalRecord(data, (*Alias)(t), &t.Overflow, turnInfoKnown, "TurnInfo")
}

// TurnError describes a turn failure.
type TurnError struct {
	Message           string `json:"message"`
	AdditionalDetails string `json:"additionalDetails,omitzero"`
	Overflow
}

var turnErrorKnown = makeSet("message", "codexErrorInfo", "additionalDetails")

// UnmarshalJSON implements json.Unmarshaler.
func (e *TurnError) UnmarshalJSON(data []byte) error {
	type Alias TurnError
	return unmarshalRecord(data, (*Alias)(e), &e.Overflow, turnErrorKnown, "TurnError")
}

// ---------- Item envelope ----------

// ItemParams holds the params for item/started, item/completed, and
// item/updated notifications. Item is raw JSON dispatched by ItemHeader.Type.
type ItemParams struct {
	Item     json.RawMessage `json:"item"`
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	Overflow
}

var itemParamsKnown = makeSet("item", "threadId", "turnId")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ItemParams) UnmarshalJSON(data []byte) error {
	type Alias ItemParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, itemParamsKnown, "ItemParams")
}

// ItemHeader extracts the discriminant fields from a raw item for dispatch.
type ItemHeader struct {
	ID   string `json:"id"`
	Type string `json:"type"`
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
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, itemDeltaParamsKnown, "ItemDeltaParams")
}

// ---------- Per-item-type structs ----------

// AgentMessageItem is an agent text response item.
type AgentMessageItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Text   string `json:"text,omitzero"`
	Phase  string `json:"phase,omitzero"`
	Status string `json:"status,omitzero"`
	Overflow
}

var agentMessageItemKnown = makeSet("id", "type", "text", "phase", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *AgentMessageItem) UnmarshalJSON(data []byte) error {
	type Alias AgentMessageItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, agentMessageItemKnown, "AgentMessageItem")
}

// PlanItem is an agent plan item.
type PlanItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Text   string `json:"text,omitzero"`
	Status string `json:"status,omitzero"`
	Overflow
}

var planItemKnown = makeSet("id", "type", "text", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *PlanItem) UnmarshalJSON(data []byte) error {
	type Alias PlanItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, planItemKnown, "PlanItem")
}

// ReasoningItem is an agent reasoning/thinking item.
type ReasoningItem struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Summary []string        `json:"summary,omitzero"`
	Content json.RawMessage `json:"content,omitzero"`
	Status  string          `json:"status,omitzero"`
	Overflow
}

var reasoningItemKnown = makeSet("id", "type", "summary", "content", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *ReasoningItem) UnmarshalJSON(data []byte) error {
	type Alias ReasoningItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, reasoningItemKnown, "ReasoningItem")
}

// CommandExecutionItem is a shell command execution item.
type CommandExecutionItem struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Command          string          `json:"command,omitzero"`
	Cwd              string          `json:"cwd,omitzero"`
	ProcessID        string          `json:"processId,omitzero"`
	Status           string          `json:"status,omitzero"`
	CommandActions   json.RawMessage `json:"commandActions,omitzero"`
	AggregatedOutput *string         `json:"aggregatedOutput,omitzero"`
	ExitCode         *int            `json:"exitCode,omitzero"`
	DurationMs       *int64          `json:"durationMs,omitzero"`
	Overflow
}

var commandExecutionItemKnown = makeSet(
	"id", "type", "command", "cwd", "processId", "status",
	"commandActions", "aggregatedOutput", "exitCode", "durationMs",
)

// UnmarshalJSON implements json.Unmarshaler.
func (item *CommandExecutionItem) UnmarshalJSON(data []byte) error {
	type Alias CommandExecutionItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, commandExecutionItemKnown, "CommandExecutionItem")
}

// FileChangeItem is a file creation/modification/deletion item.
type FileChangeItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Changes []FileUpdateChange `json:"changes,omitzero"`
	Status  string             `json:"status,omitzero"`
	Overflow
}

var fileChangeItemKnown = makeSet("id", "type", "changes", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *FileChangeItem) UnmarshalJSON(data []byte) error {
	type Alias FileChangeItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, fileChangeItemKnown, "FileChangeItem")
}

// McpToolCallItem is an MCP tool call item.
type McpToolCallItem struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Server     string             `json:"server,omitzero"`
	Tool       string             `json:"tool,omitzero"`
	Status     string             `json:"status,omitzero"`
	Arguments  json.RawMessage    `json:"arguments,omitzero"`
	Result     *McpToolCallResult `json:"result,omitzero"`
	Error      *McpToolCallError  `json:"error,omitzero"`
	DurationMs *int64             `json:"durationMs,omitzero"`
	Overflow
}

var mcpToolCallItemKnown = makeSet(
	"id", "type", "server", "tool", "status",
	"arguments", "result", "error", "durationMs",
)

// UnmarshalJSON implements json.Unmarshaler.
func (item *McpToolCallItem) UnmarshalJSON(data []byte) error {
	type Alias McpToolCallItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, mcpToolCallItemKnown, "McpToolCallItem")
}

// DynamicToolCallItem is a dynamically registered tool call item.
type DynamicToolCallItem struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Tool         string          `json:"tool,omitzero"`
	Arguments    json.RawMessage `json:"arguments,omitzero"`
	Status       string          `json:"status,omitzero"`
	ContentItems json.RawMessage `json:"contentItems,omitzero"`
	Success      *bool           `json:"success,omitzero"`
	DurationMs   *int64          `json:"durationMs,omitzero"`
	Overflow
}

var dynamicToolCallItemKnown = makeSet(
	"id", "type", "tool", "arguments", "status",
	"contentItems", "success", "durationMs",
)

// UnmarshalJSON implements json.Unmarshaler.
func (item *DynamicToolCallItem) UnmarshalJSON(data []byte) error {
	type Alias DynamicToolCallItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, dynamicToolCallItemKnown, "DynamicToolCallItem")
}

// CollabAgentToolCallItem is a collaborative multi-agent tool call item.
type CollabAgentToolCallItem struct {
	ID                string          `json:"id"`
	Type              string          `json:"type"`
	Tool              string          `json:"tool,omitzero"`
	Status            string          `json:"status,omitzero"`
	SenderThreadID    string          `json:"senderThreadId,omitzero"`
	ReceiverThreadIDs json.RawMessage `json:"receiverThreadIds,omitzero"`
	Prompt            string          `json:"prompt,omitzero"`
	AgentsStates      json.RawMessage `json:"agentsStates,omitzero"`
	Overflow
}

var collabAgentToolCallItemKnown = makeSet(
	"id", "type", "tool", "status",
	"senderThreadId", "receiverThreadIds", "prompt", "agentsStates",
)

// UnmarshalJSON implements json.Unmarshaler.
func (item *CollabAgentToolCallItem) UnmarshalJSON(data []byte) error {
	type Alias CollabAgentToolCallItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, collabAgentToolCallItemKnown, "CollabAgentToolCallItem")
}

// WebSearchItem is a web search item.
type WebSearchItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Query  string `json:"query,omitzero"`
	Action string `json:"action,omitzero"`
	Status string `json:"status,omitzero"`
	Overflow
}

var webSearchItemKnown = makeSet("id", "type", "query", "action", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *WebSearchItem) UnmarshalJSON(data []byte) error {
	type Alias WebSearchItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, webSearchItemKnown, "WebSearchItem")
}

// ImageViewItem is an image viewing item.
type ImageViewItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Path   string `json:"path,omitzero"`
	Status string `json:"status,omitzero"`
	Overflow
}

var imageViewItemKnown = makeSet("id", "type", "path", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *ImageViewItem) UnmarshalJSON(data []byte) error {
	type Alias ImageViewItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, imageViewItemKnown, "ImageViewItem")
}

// EnteredReviewModeItem signals the agent entered review mode.
type EnteredReviewModeItem struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Review json.RawMessage `json:"review,omitzero"`
	Overflow
}

var enteredReviewModeItemKnown = makeSet("id", "type", "review")

// UnmarshalJSON implements json.Unmarshaler.
func (item *EnteredReviewModeItem) UnmarshalJSON(data []byte) error {
	type Alias EnteredReviewModeItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, enteredReviewModeItemKnown, "EnteredReviewModeItem")
}

// ExitedReviewModeItem signals the agent exited review mode.
type ExitedReviewModeItem struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Review json.RawMessage `json:"review,omitzero"`
	Overflow
}

var exitedReviewModeItemKnown = makeSet("id", "type", "review")

// UnmarshalJSON implements json.Unmarshaler.
func (item *ExitedReviewModeItem) UnmarshalJSON(data []byte) error {
	type Alias ExitedReviewModeItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, exitedReviewModeItemKnown, "ExitedReviewModeItem")
}

// ContextCompactionItem signals a context window compaction.
type ContextCompactionItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Overflow
}

var contextCompactionItemKnown = makeSet("id", "type")

// UnmarshalJSON implements json.Unmarshaler.
func (item *ContextCompactionItem) UnmarshalJSON(data []byte) error {
	type Alias ContextCompactionItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, contextCompactionItemKnown, "ContextCompactionItem")
}

// UserMessageItem is a user-submitted message item.
type UserMessageItem struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitzero"`
	Status  string          `json:"status,omitzero"`
	Overflow
}

var userMessageItemKnown = makeSet("id", "type", "content", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (item *UserMessageItem) UnmarshalJSON(data []byte) error {
	type Alias UserMessageItem
	return unmarshalRecord(data, (*Alias)(item), &item.Overflow, userMessageItemKnown, "UserMessageItem")
}

// ---------- Item field types ----------

// FileUpdateChange describes a single file change within a fileChange item.
type FileUpdateChange struct {
	Path string          `json:"path"`
	Kind PatchChangeKind `json:"kind"`
	Diff string          `json:"diff,omitzero"`
}

// PatchChangeKind is the discriminated kind for FileUpdateChange.
type PatchChangeKind struct {
	Type     string  `json:"type"`
	MovePath *string `json:"movePath,omitzero"`
}

// McpToolCallResult holds the result of a successful MCP tool call.
type McpToolCallResult struct {
	Content           []json.RawMessage `json:"content"`
	StructuredContent json.RawMessage   `json:"structuredContent,omitzero"`
}

// McpToolCallError holds the error from a failed MCP tool call.
type McpToolCallError struct {
	Message string `json:"message"`
}

// ---------- Token usage ----------

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
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, tokenUsageUpdatedParamsKnown, "TokenUsageUpdatedParams")
}

// ThreadTokenUsage holds cumulative and per-turn token usage for a thread.
type ThreadTokenUsage struct {
	Total              TokenUsageBreakdown `json:"total"`
	Last               TokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64              `json:"modelContextWindow,omitzero"`
}

// TokenUsageBreakdown contains a detailed breakdown of token counts.
type TokenUsageBreakdown struct {
	TotalTokens           int64 `json:"totalTokens"`
	InputTokens           int64 `json:"inputTokens"`
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
}

// ---------- Delta notification params ----------

// CommandOutputDeltaParams holds params for item/commandExecution/outputDelta.
type CommandOutputDeltaParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
	Overflow
}

var commandOutputDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta")

// UnmarshalJSON implements json.Unmarshaler.
func (p *CommandOutputDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias CommandOutputDeltaParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, commandOutputDeltaParamsKnown, "CommandOutputDeltaParams")
}

// TerminalInteractionParams holds params for item/commandExecution/terminalInteraction.
type TerminalInteractionParams struct {
	ThreadID  string `json:"threadId"`
	TurnID    string `json:"turnId"`
	ItemID    string `json:"itemId"`
	ProcessID string `json:"processId"`
	Stdin     string `json:"stdin"`
	Overflow
}

var terminalInteractionParamsKnown = makeSet("threadId", "turnId", "itemId", "processId", "stdin")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TerminalInteractionParams) UnmarshalJSON(data []byte) error {
	type Alias TerminalInteractionParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, terminalInteractionParamsKnown, "TerminalInteractionParams")
}

// FileChangeOutputDeltaParams holds params for item/fileChange/outputDelta.
type FileChangeOutputDeltaParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
	Overflow
}

var fileChangeOutputDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta")

// UnmarshalJSON implements json.Unmarshaler.
func (p *FileChangeOutputDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias FileChangeOutputDeltaParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, fileChangeOutputDeltaParamsKnown, "FileChangeOutputDeltaParams")
}

// ReasoningSummaryTextDeltaParams holds params for item/reasoning/summaryTextDelta.
type ReasoningSummaryTextDeltaParams struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	SummaryIndex int    `json:"summaryIndex"`
	Overflow
}

var reasoningSummaryTextDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta", "summaryIndex")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ReasoningSummaryTextDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias ReasoningSummaryTextDeltaParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, reasoningSummaryTextDeltaParamsKnown, "ReasoningSummaryTextDeltaParams")
}

// ReasoningSummaryPartAddedParams holds params for item/reasoning/summaryPartAdded.
type ReasoningSummaryPartAddedParams struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	SummaryIndex int    `json:"summaryIndex"`
	Overflow
}

var reasoningSummaryPartAddedParamsKnown = makeSet("threadId", "turnId", "itemId", "summaryIndex")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ReasoningSummaryPartAddedParams) UnmarshalJSON(data []byte) error {
	type Alias ReasoningSummaryPartAddedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, reasoningSummaryPartAddedParamsKnown, "ReasoningSummaryPartAddedParams")
}

// ReasoningTextDeltaParams holds params for item/reasoning/textDelta.
type ReasoningTextDeltaParams struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	ContentIndex int    `json:"contentIndex"`
	Overflow
}

var reasoningTextDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta", "contentIndex")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ReasoningTextDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias ReasoningTextDeltaParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, reasoningTextDeltaParamsKnown, "ReasoningTextDeltaParams")
}

// PlanDeltaParams holds params for item/plan/delta.
type PlanDeltaParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
	Overflow
}

var planDeltaParamsKnown = makeSet("threadId", "turnId", "itemId", "delta")

// UnmarshalJSON implements json.Unmarshaler.
func (p *PlanDeltaParams) UnmarshalJSON(data []byte) error {
	type Alias PlanDeltaParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, planDeltaParamsKnown, "PlanDeltaParams")
}

// McpToolCallProgressParams holds params for item/mcpToolCall/progress.
type McpToolCallProgressParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Message  string `json:"message"`
	Overflow
}

var mcpToolCallProgressParamsKnown = makeSet("threadId", "turnId", "itemId", "message")

// UnmarshalJSON implements json.Unmarshaler.
func (p *McpToolCallProgressParams) UnmarshalJSON(data []byte) error {
	type Alias McpToolCallProgressParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, mcpToolCallProgressParamsKnown, "McpToolCallProgressParams")
}

// ---------- Other notification params ----------

// TurnDiffUpdatedParams holds params for turn/diff/updated.
type TurnDiffUpdatedParams struct {
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	Diff     json.RawMessage `json:"diff"`
	Overflow
}

var turnDiffUpdatedParamsKnown = makeSet("threadId", "turnId", "diff")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TurnDiffUpdatedParams) UnmarshalJSON(data []byte) error {
	type Alias TurnDiffUpdatedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, turnDiffUpdatedParamsKnown, "TurnDiffUpdatedParams")
}

// TurnPlanUpdatedParams holds params for turn/plan/updated.
type TurnPlanUpdatedParams struct {
	ThreadID    string          `json:"threadId"`
	TurnID      string          `json:"turnId"`
	Explanation string          `json:"explanation,omitzero"`
	Plan        json.RawMessage `json:"plan,omitzero"`
	Overflow
}

var turnPlanUpdatedParamsKnown = makeSet("threadId", "turnId", "explanation", "plan")

// UnmarshalJSON implements json.Unmarshaler.
func (p *TurnPlanUpdatedParams) UnmarshalJSON(data []byte) error {
	type Alias TurnPlanUpdatedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, turnPlanUpdatedParamsKnown, "TurnPlanUpdatedParams")
}

// ThreadStatusChangedParams holds params for thread/status/changed.
type ThreadStatusChangedParams struct {
	ThreadID string `json:"threadId"`
	Status   string `json:"status"`
	Overflow
}

var threadStatusChangedParamsKnown = makeSet("threadId", "status")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ThreadStatusChangedParams) UnmarshalJSON(data []byte) error {
	type Alias ThreadStatusChangedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, threadStatusChangedParamsKnown, "ThreadStatusChangedParams")
}

// ThreadNameUpdatedParams holds params for thread/name/updated.
type ThreadNameUpdatedParams struct {
	ThreadID   string `json:"threadId"`
	ThreadName string `json:"threadName"`
	Overflow
}

var threadNameUpdatedParamsKnown = makeSet("threadId", "threadName")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ThreadNameUpdatedParams) UnmarshalJSON(data []byte) error {
	type Alias ThreadNameUpdatedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, threadNameUpdatedParamsKnown, "ThreadNameUpdatedParams")
}

// ModelReroutedParams holds params for model/rerouted.
type ModelReroutedParams struct {
	ThreadID  string `json:"threadId"`
	TurnID    string `json:"turnId"`
	FromModel string `json:"fromModel"`
	ToModel   string `json:"toModel"`
	Reason    string `json:"reason,omitzero"`
	Overflow
}

var modelReroutedParamsKnown = makeSet("threadId", "turnId", "fromModel", "toModel", "reason")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ModelReroutedParams) UnmarshalJSON(data []byte) error {
	type Alias ModelReroutedParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, modelReroutedParamsKnown, "ModelReroutedParams")
}

// ErrorNotificationParams holds params for error notifications.
type ErrorNotificationParams struct {
	Error     *TurnError `json:"error"`
	WillRetry bool       `json:"willRetry,omitzero"`
	ThreadID  string     `json:"threadId,omitzero"`
	TurnID    string     `json:"turnId,omitzero"`
	Overflow
}

var errorNotificationParamsKnown = makeSet("error", "willRetry", "threadId", "turnId")

// UnmarshalJSON implements json.Unmarshaler.
func (p *ErrorNotificationParams) UnmarshalJSON(data []byte) error {
	type Alias ErrorNotificationParams
	return unmarshalRecord(data, (*Alias)(p), &p.Overflow, errorNotificationParamsKnown, "ErrorNotificationParams")
}
