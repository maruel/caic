package codex

import (
	"encoding/json"
	"fmt"

	"github.com/maruel/caic/backend/internal/agent"
)

// ParseMessage decodes a single Codex CLI exec --json line into a typed
// agent.Message.
func ParseMessage(line []byte) (agent.Message, error) {
	var rec Record
	if err := json.Unmarshal(line, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal record: %w", err)
	}
	switch rec.Type {
	case TypeThreadStarted:
		r, err := rec.AsThreadStarted()
		if err != nil {
			return nil, err
		}
		return &agent.SystemInitMessage{
			MessageType: "system",
			Subtype:     "init",
			SessionID:   r.ThreadID,
		}, nil

	case TypeTurnStarted:
		return &agent.SystemMessage{
			MessageType: "system",
			Subtype:     "turn_started",
		}, nil

	case TypeTurnCompleted:
		r, err := rec.AsTurnCompleted()
		if err != nil {
			return nil, err
		}
		return &agent.ResultMessage{
			MessageType: "result",
			Subtype:     "result",
			Usage: agent.Usage{
				InputTokens:          r.Usage.InputTokens,
				OutputTokens:         r.Usage.OutputTokens,
				CacheReadInputTokens: r.Usage.CachedInputTokens,
			},
		}, nil

	case TypeTurnFailed:
		r, err := rec.AsTurnFailed()
		if err != nil {
			return nil, err
		}
		return &agent.ResultMessage{
			MessageType: "result",
			Subtype:     "result",
			IsError:     true,
			Result:      r.Error,
		}, nil

	case TypeItemStarted:
		return parseItemStarted(rec)

	case TypeItemCompleted:
		return parseItemCompleted(rec)

	case TypeItemUpdated:
		return &agent.RawMessage{MessageType: rec.Type, Raw: append([]byte(nil), line...)}, nil

	case TypeError:
		return &agent.RawMessage{MessageType: rec.Type, Raw: append([]byte(nil), line...)}, nil

	case "caic_diff_stat":
		var m agent.DiffStatMessage
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, err
		}
		return &m, nil

	default:
		return &agent.RawMessage{MessageType: rec.Type, Raw: append([]byte(nil), line...)}, nil
	}
}

// parseItemStarted handles item.started events. For command_execution, emits
// an AssistantMessage with a tool_use block. Other item types are passed
// through as RawMessages.
func parseItemStarted(rec Record) (agent.Message, error) {
	r, err := rec.AsItem()
	if err != nil {
		return nil, err
	}
	switch r.Item.Type {
	case ItemTypeCommandExecution:
		input, _ := json.Marshal(map[string]string{"command": r.Item.Command})
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type:  "tool_use",
					ID:    r.Item.ID,
					Name:  "Bash",
					Input: input,
				}},
			},
		}, nil

	case ItemTypeMCPToolCall:
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type:  "tool_use",
					ID:    r.Item.ID,
					Name:  r.Item.Tool,
					Input: r.Item.Arguments,
				}},
			},
		}, nil

	default:
		return &agent.RawMessage{MessageType: rec.Type, Raw: append([]byte(nil), rec.Raw()...)}, nil
	}
}

// parseItemCompleted handles item.completed events. Dispatches based on the
// item type to produce the appropriate agent.Message.
func parseItemCompleted(rec Record) (agent.Message, error) {
	r, err := rec.AsItem()
	if err != nil {
		return nil, err
	}
	switch r.Item.Type {
	case ItemTypeAgentMessage:
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type: "text",
					Text: r.Item.Text,
				}},
			},
		}, nil

	case ItemTypeReasoning:
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type: "text",
					Text: r.Item.Text,
				}},
			},
		}, nil

	case ItemTypeCommandExecution:
		raw, _ := json.Marshal(r.Item.AggregatedOutput)
		return &agent.UserMessage{
			MessageType:     "user",
			Message:         raw,
			ParentToolUseID: &r.Item.ID,
		}, nil

	case ItemTypeFileChange:
		toolName := "Edit"
		for _, c := range r.Item.Changes {
			if c.Kind == "add" {
				toolName = "Write"
				break
			}
		}
		input, _ := json.Marshal(r.Item.Changes)
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type:  "tool_use",
					ID:    r.Item.ID,
					Name:  toolName,
					Input: input,
				}},
			},
		}, nil

	case ItemTypeMCPToolCall:
		content := r.Item.Result
		if r.Item.Error != "" {
			content = r.Item.Error
		}
		raw, _ := json.Marshal(content)
		return &agent.UserMessage{
			MessageType:     "user",
			Message:         raw,
			ParentToolUseID: &r.Item.ID,
		}, nil

	case ItemTypeWebSearch:
		input, _ := json.Marshal(map[string]string{"query": r.Item.Query})
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type:  "tool_use",
					ID:    r.Item.ID,
					Name:  "WebSearch",
					Input: input,
				}},
			},
		}, nil

	case ItemTypeTodoList:
		input, _ := json.Marshal(r.Item.Items)
		return &agent.AssistantMessage{
			MessageType: "assistant",
			Message: agent.APIMessage{
				Role: "assistant",
				Content: []agent.ContentBlock{{
					Type:  "tool_use",
					ID:    r.Item.ID,
					Name:  "TodoWrite",
					Input: input,
				}},
			},
		}, nil

	case ItemTypeError:
		return &agent.RawMessage{MessageType: "error", Raw: append([]byte(nil), rec.Raw()...)}, nil

	default:
		return &agent.RawMessage{MessageType: rec.Type, Raw: append([]byte(nil), rec.Raw()...)}, nil
	}
}
