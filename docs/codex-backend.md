# Codex CLI Backend — Integration Spec

Add OpenAI Codex CLI as a third `agent.Backend` alongside Claude and Gemini.

## Codex CLI Overview

Codex CLI is OpenAI's agentic coding tool (Rust binary, npm-installable).
Non-interactive mode: `codex exec --json "prompt"`. The `--json` flag emits
NDJSON (one JSON object per line) on stdout; human progress goes to stderr.

Repo: https://github.com/openai/codex

## Launch Command

```
codex exec --json \
  --full-auto \              # equivalent to --dangerously-skip-permissions / --yolo
  -m <model> \               # e.g. o4-mini, codex-mini-latest
  "prompt"
```

Key flags:
| Flag | Purpose |
|------|---------|
| `--json` | NDJSON event stream on stdout |
| `--full-auto` | Skip all approval prompts |
| `-m <model>` | Model selection |
| `--ephemeral` | Don't persist session files |

No `--resume` equivalent — Codex sessions are ephemeral by default.

## Stdin

Codex `exec` mode is fire-and-forget. It does not read follow-up prompts
from stdin (unlike Claude's `-p` NDJSON stdin or Gemini's plain-text stdin).
`WritePrompt` should return an error or be a no-op.

If interactive follow-ups are needed, use `codex app-server` (JSON-RPC over
stdio) instead — but that is a significantly more complex protocol and likely
not worth it for v1.

## Streaming JSON Protocol (`codex exec --json`)

### Envelope

Every line has a `"type"` field:

```json
{"type":"thread.started","thread_id":"0199a213-..."}
```

### Event Types

| Type | When | Key Fields |
|------|------|------------|
| `thread.started` | Session begins | `thread_id` |
| `turn.started` | New agent turn begins | — |
| `turn.completed` | Turn ends | `usage` (tokens) |
| `turn.failed` | Turn errors | `error` |
| `item.started` | Tool/action begins | `item` |
| `item.updated` | Incremental update | `item` |
| `item.completed` | Tool/action finishes | `item` |
| `error` | Non-fatal error | `message` |

### Item Types (inside `item.started` / `item.completed`)

Each item has `item.type`:

| `item.type` | Description | Key Fields |
|-------------|-------------|------------|
| `agent_message` | Final text response | `text` |
| `reasoning` | Model thinking summary | `text` |
| `command_execution` | Shell command | `command`, `aggregated_output`, `exit_code`, `status` |
| `file_change` | File write/edit/delete | `changes[].path`, `changes[].kind` (add/update/delete) |
| `mcp_tool_call` | MCP tool invocation | `server`, `tool`, `arguments`, `result`, `error` |
| `web_search` | Web search | `query` |
| `todo_list` | Plan tracking | `items[].text`, `items[].completed` |
| `error` | Non-fatal warning | `message` |

### Status Values

`status` on items: `in_progress` | `completed` | `failed`.

### Example Stream

```jsonl
{"type":"thread.started","thread_id":"0199a213-81c0-7800-8aa1-bbab2a035a53"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"**Scanning...**"}}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"docs\nsrc\n","exit_code":0,"status":"completed"}}
{"type":"item.completed","item":{"id":"item_4","type":"file_change","changes":[{"path":"docs/foo.md","kind":"add"}],"status":"completed"}}
{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Done."}}
{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122}}
```

### Usage Object (on `turn.completed`)

```json
{
  "input_tokens": 24763,
  "cached_input_tokens": 24448,
  "output_tokens": 122
}
```

Codex does not report `total_cost_usd`. Cost must be computed externally or
left at 0.

## Mapping to `agent.Message`

### Record Types → Go Structs

Following the Gemini pattern (`record.go` with `Overflow` for forward compat):

| Codex Event | Go Record Type |
|-------------|---------------|
| `thread.started` | `ThreadStartedRecord` → `agent.SystemInitMessage` |
| `turn.started` | `TurnStartedRecord` → `agent.SystemMessage` (subtype "turn_started") |
| `turn.completed` | `TurnCompletedRecord` → `agent.ResultMessage` |
| `turn.failed` | `TurnFailedRecord` → `agent.ResultMessage` (is_error=true) |
| `item.completed` + `agent_message` | → `agent.AssistantMessage` (text block) |
| `item.completed` + `reasoning` | → `agent.AssistantMessage` (text block) or `agent.RawMessage` |
| `item.started` + `command_execution` | → `agent.AssistantMessage` (tool_use block, name="Bash") |
| `item.completed` + `command_execution` | → `agent.UserMessage` (tool result) |
| `item.completed` + `file_change` | → `agent.AssistantMessage` (tool_use, name="Write"/"Edit") |
| `item.started` + `mcp_tool_call` | → `agent.AssistantMessage` (tool_use) |
| `item.completed` + `mcp_tool_call` | → `agent.UserMessage` (tool result) |
| `item.updated` | → `agent.RawMessage` (pass-through) |
| `error` | → `agent.RawMessage` |

### Tool Name Mapping

Codex doesn't expose individual tool names for its built-in tools the way
Claude/Gemini do. Instead, actions surface as typed items. Map item types:

```go
// Item type → normalized tool name.
var itemTypeToTool = map[string]string{
    "command_execution": "Bash",
    "file_change":       "Edit",  // or "Write" based on changes[].kind
    "web_search":        "WebSearch",
    "todo_list":         "TodoWrite",
}
```

For `mcp_tool_call` items, use the `tool` field directly (MCP tool names are
already provider-agnostic).

### Key Difference: Two-Phase Items

Unlike Claude/Gemini (which emit separate `tool_use` + `tool_result` records),
Codex emits `item.started` (tool invoked) + `item.completed` (result ready)
for the **same item ID**. The parser must:

1. On `item.started` with `command_execution`: emit `AssistantMessage` with a
   `tool_use` content block (tool ID = item ID, name = "Bash", input =
   `{"command": item.command}`).
2. On `item.completed` with `command_execution`: emit `UserMessage` with the
   tool result (parent_tool_use_id = item ID, content = aggregated_output).

For `file_change`, only `item.completed` is emitted (no started phase). Emit
both the tool_use and tool_result together, or just the tool_use with the
changes as input.

## Implementation Checklist

### 1. `agent/types.go`

Add `Codex Harness = "codex"` constant.

### 2. `agent/codex/` package

| File | Contents |
|------|----------|
| `codex.go` | `Backend` struct implementing `agent.Backend`. `Harness() → "codex"`. `buildArgs()` constructs `codex exec --json --full-auto`. `WritePrompt` returns error (exec mode is non-interactive). |
| `record.go` | `Record` envelope + typed records: `ThreadStartedRecord`, `TurnStartedRecord`, `TurnCompletedRecord`, `TurnFailedRecord`, `ItemRecord`. All embed `Overflow`. |
| `parse.go` | `ParseMessage(line []byte) (agent.Message, error)`. Two-level dispatch: outer `type` field, then `item.type` for item events. Tool name mapping. |
| `unknown.go` | Copy from `agent/gemini/unknown.go` (or extract to shared package). |
| `parse_test.go` | Test cases from the example stream above. |
| `record_test.go` | Round-trip and unknown-field tests. |

### 3. `task/runner.go`

Register: `agent.Codex: &codex.Backend{}` in `Backends` map.

### 4. `task/load.go`

Add `case agent.Codex: return agentcodex.ParseMessage` in `parseFnForHarness`.

### 5. `dto/types.go`

Add `HarnessCodex Harness = "codex"`.

### 6. Frontend

No changes needed — the harness selector (`GET /api/v1/harnesses`) is dynamic.

## Risks and Constraints

- **No follow-up prompts.** `codex exec` is single-shot. The `SendInput`
  path in `task.Task` won't work. Either: (a) document this limitation, or
  (b) switch to `codex app-server` JSON-RPC for interactive use.
- **No session resume.** Codex exec is ephemeral. `ResumeSessionID` is
  ignored.
- **No cost reporting.** `TotalCostUSD` will be 0. Token counts are available.
- **Relay compatibility.** The relay daemon (relay.py) bridges stdin/stdout and
  is process-agnostic. It should work unchanged with `codex exec --json`.
  However, since stdin is unused, the relay's stdin forwarding is a no-op.
- **`item.started` + `item.completed` pairing.** Unlike the other backends
  where tool_use and tool_result are independent records, Codex pairs them by
  item ID. The parser must handle this correctly to avoid duplicate or orphaned
  tool events in the frontend.
- **No `--resume` flag.** `AttachRelay` can still reconnect to a running relay,
  but cannot resume a terminated Codex session.
