# Multi-Agent Abstraction Plan for CAIC

## Feasibility: Gemini CLI

**Verdict: Yes, feasible.** Gemini CLI supports:
- Headless mode: `gemini -p "prompt"` (same pattern as Claude's `-p`)
- Streaming JSON: `--output-format stream-json` (NDJSON, same concept)
- Session resume: `--resume [UUID]`
- Auto-approve: `--yolo` (equivalent to `--dangerously-skip-permissions`)
- Model selection: `-m gemini-2.5-flash` etc.

**Key differences** that affect the abstraction:
1. **Wire format differs** — Gemini's `stream-json` events have a different schema than Claude Code's NDJSON messages (different type names, different content block structure)
2. **No SDK** — Gemini CLI is the only programmatic interface (same as Claude Code in practice: both are subprocess-based)
3. **Tool names differ** — `read_file` vs `Read`, `run_shell_command` vs `Bash`, etc.
4. **Result/stats schema differs** — Gemini reports `stats.models`/`stats.tools` vs Claude's `ResultMessage` with `total_cost_usd`/`usage`
5. **Session management** — Gemini stores sessions in `~/.gemini/tmp/`, Claude in its own location. Both support `--resume`.

## Current Claude-Coupled Code

Three layers are coupled to Claude Code's specific protocol:

### Layer 1: Process Launch (`agent/agent.go`)
- `Start()`, `StartWithRelay()` hardcode `claude -p --input-format stream-json --output-format stream-json --verbose --dangerously-skip-permissions`
- Input format: `{"type":"user","message":{"role":"user","content":"..."}}`
- The relay.py is Claude-agnostic (it just bridges stdin/stdout/socket) — **reusable as-is**

### Layer 2: Message Parsing (`agent/types.go`, `agent/agent.go:ParseMessage`)
- `Message` interface + concrete types (`AssistantMessage`, `UserMessage`, `ResultMessage`, etc.)
- `ParseMessage()` decodes Claude Code's specific NDJSON envelope (`type`, `subtype`)
- Content blocks assume Anthropic API format (`text`, `tool_use` with `id`/`name`/`input`)

### Layer 3: Event Conversion (`server/eventconv.go`)
- `convertMessage()` maps `agent.Message` → `dto.EventMessage`
- Hardcodes Claude-specific tool names: `"AskUserQuestion"`, `"TodoWrite"`
- Extracts Claude-specific fields: `parent_tool_use_id`, `Usage` structure

### Layer 4: Task State Machine (`task/task.go`)
- `addMessage()` inspects `*agent.SystemInitMessage` for `SessionID`, `Model`, `ClaudeCodeVersion`
- `lastAssistantHasAsk()` checks for `"AskUserQuestion"` tool name
- `RestoreMessages()` relies on Claude message types

## Proposed Abstraction

### Principle: Normalize at the boundary

Each agent backend translates its native wire format into a **shared internal message model** (the existing `agent.Message` types). The rest of the system (task, eventconv, SSE, frontend) remains unchanged.

### Architecture

```
┌─────────────────┐     ┌─────────────────┐
│  Claude Backend │     │  Gemini Backend  │     (future: Kilo, Aider, ...)
│                 │     │                  │
│  Launches       │     │  Launches        │
│  claude -p ...  │     │  gemini -p ...   │
│                 │     │                  │
│  Parses Claude  │     │  Parses Gemini   │
│  NDJSON format  │     │  NDJSON format   │
│                 │     │                  │
│  Emits          │     │  Emits           │
│  agent.Message  │     │  agent.Message   │
└────────┬────────┘     └────────┬─────────┘
         │                       │
         └───────────┬───────────┘
                     │
              agent.Message (shared)
                     │
         ┌───────────┴───────────┐
         │  task / eventconv /   │
         │  SSE / frontend       │
         │  (unchanged)          │
         └───────────────────────┘
```

### Interface Design

```go
// Package agent

// Backend launches and communicates with a coding agent process.
type Backend interface {
    // Start launches the agent in the given container. Messages are emitted
    // to msgCh as agent.Message (normalized). logW receives raw wire-format
    // lines for debugging/replay.
    Start(ctx context.Context, opts Options, msgCh chan<- Message, logW io.Writer) (*Session, error)

    // WritePrompt sends a user prompt to the running session.
    WritePrompt(session *Session, prompt string, logW io.Writer) error

    // ParseMessage decodes a single wire-format line into a normalized Message.
    // Used for log replay (load.go).
    ParseMessage(line []byte) (Message, error)

    // Name returns a human-readable backend name ("claude", "gemini", etc.)
    Name() string
}
```

**Why this shape:**
- `Start` replaces the current `AgentStartFn` signature — same contract but now includes the wire-format translation
- `WritePrompt` replaces `writeMessage()` — each backend knows its own stdin format
- `ParseMessage` is needed for log replay (currently `agent.ParseMessage` is called from `task/load.go`)
- The relay.py infrastructure stays in the Claude backend since it's process-generic

### What changes per package

#### `agent/` package — split into sub-packages

```
agent/
  message.go       — Message interface + shared types (unchanged)
  session.go       — Session struct (unchanged, process-agnostic)
  backend.go       — Backend interface definition
  claude/
    claude.go       — Start, WritePrompt, ParseMessage for Claude Code
    relay.go        — relay.py deployment (moved from agent.go)
  gemini/
    gemini.go       — Start, WritePrompt, ParseMessage for Gemini CLI
    translate.go    — Gemini stream-json → agent.Message translation
```

#### `task/runner.go` — minimal changes
- `AgentStartFn` field → `AgentBackend agent.Backend` field
- Calls `r.AgentBackend.Start(...)` instead of `r.AgentStartFn(...)`
- Calls `r.AgentBackend.WritePrompt(session, prompt, logW)` instead of `session.Send(prompt)`
- Log replay calls `r.AgentBackend.ParseMessage(line)` instead of `agent.ParseMessage(line)`

#### `task/task.go` — minor changes
- `SendInput()` needs access to the backend's `WritePrompt` — either stored on Task or passed through
- `addMessage()` — the Claude-specific field extraction (`SessionID`, `Model`, `ClaudeCodeVersion`) already works through the `Message` interface; Gemini backend would populate `SystemInitMessage` with equivalent data
- `lastAssistantHasAsk()` — works via `ContentBlock.Name == "AskUserQuestion"`. Gemini's equivalent tool is `ask_user`. The **Gemini backend's translator** maps `ask_user` → `AskUserQuestion` in the normalized message, so this code stays unchanged.

#### `server/eventconv.go` — unchanged
- Already works with `agent.Message` interface. As long as backends normalize to the same message types, no changes needed.

#### `dto/types.go`, `dto/events.go` — unchanged
- Agent-agnostic. They define the SSE/API contract, not the wire format.

#### `server/server.go` — add backend selection
- Task creation request gets a new optional `agent` field (default: `"claude"`)
- Server holds a `map[string]agent.Backend` registry
- Runner selection uses the backend matching the requested agent

#### Frontend — minimal changes
- Task creation form gets an agent selector dropdown
- `EventInit` already has `Model` and `ClaudeCodeVersion`; rename `ClaudeCodeVersion` → `AgentVersion` in dto
- Display the agent name in the task header

### Gemini Backend: Translation Rules

| Gemini stream-json event | Normalized agent.Message |
|---|---|
| Session start / model info | `SystemInitMessage` (populate Model, tools list) |
| Text output | `AssistantMessage` with `ContentBlock{Type:"text"}` |
| Tool call (e.g. `read_file`) | `AssistantMessage` with `ContentBlock{Type:"tool_use", Name:"Read"}` |
| Tool result | `UserMessage` with `ParentToolUseID` |
| Final stats / completion | `ResultMessage` (map costs/tokens/turns) |

Tool name mapping table:
| Gemini | Normalized (Claude names) |
|---|---|
| `read_file` / `read_many_files` | `Read` |
| `write_file` | `Write` |
| `replace` | `Edit` |
| `run_shell_command` | `Bash` |
| `grep` | `Grep` |
| `glob` | `Glob` |
| `web_fetch` | `WebFetch` |
| `google_web_search` | `WebSearch` |
| `ask_user` | `AskUserQuestion` |
| `write_todos` | `TodoWrite` |

### Implementation Order

1. **Extract `Backend` interface + move Claude code into `agent/claude/`** — pure refactor, no behavior change
2. **Add `agent` field to task creation DTO** — plumb through server/runner
3. **Implement Gemini backend** — `agent/gemini/` with stream-json parsing and translation
4. **Frontend: agent selector** — dropdown in task creation, agent name in task header
5. **Rename `ClaudeCodeVersion` → `AgentVersion`** throughout dto/frontend

### Risk Assessment

- **Gemini stream-json format is under-documented.** Need to empirically capture output from `gemini -p "hello" --output-format stream-json` and reverse-engineer the event schema. The format was stabilized in v0.20+ but exact field names need validation.
- **Relay compatibility.** The relay.py is process-agnostic (stdin/stdout bridge), so it should work with Gemini CLI unchanged. The only assumption is that the child process reads stdin and writes stdout — both CLIs do this.
- **Cost tracking.** Gemini has a free tier (no cost). The `ResultMessage.TotalCostUSD` would be 0 or derived from Gemini's stats if available.
- **Session resume.** Gemini's `--resume` works differently (local file-based sessions). Need to verify it works in a container where `~/.gemini/` persists.

### Non-Goals (for now)
- MCP server integration (both CLIs support it, but not needed for the core abstraction)
- Multiple simultaneous backends in a single task
- Backend-specific settings UI (model lists, API key management)
