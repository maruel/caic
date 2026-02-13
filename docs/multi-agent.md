# Multi-Agent Abstraction Plan for CAIC

## Completed: Backend Interface + Claude Subpackage

Step 1 is done. The `agent.Backend` interface is extracted and the Claude-specific code lives in `agent/claude/`.

### What was done

- **`agent/backend.go`** — `Backend` interface with `Start`, `AttachRelay`, `ReadRelayOutput`, `ParseMessage`, `Name`
- **`agent/claude/claude.go`** — Claude Code backend: launches `claude -p` via relay, parses Claude NDJSON, writes Claude stdin format
- **`agent/agent.go`** — Now contains only shared infrastructure: `Session` (with pluggable `WriteFn`), `ParseMessage`, `Options`, relay utilities (`DeployRelay`, `IsRelayRunning`, `HasRelayDir`, `ReadPlan`), message types
- **`task/runner.go`** — `AgentStartFn func(...)` replaced with `AgentBackend agent.Backend`; defaults to `&claude.Backend{}`; added `ReadRelayOutput` convenience method
- **`server/server.go`** — `SetRunnerOps` takes `agent.Backend` instead of a start function; `adoptOne` uses `runner.ReadRelayOutput` instead of `agent.ReadRelayOutput`
- **`cmd/caic/main.go`** — `fakeAgentStart` func replaced with `fakeBackend` struct implementing `agent.Backend`

### Key design decisions

- **`WriteFn` on `Session`** instead of `WritePrompt` on `Backend`. `Session.Send(prompt)` still works, but each backend provides its own `WriteFn` to `NewSession`. This avoids changing `task.Task.SendInput` or storing the backend on the task.
- **Relay utilities stay in `agent/`** — `DeployRelay`, `IsRelayRunning`, `HasRelayDir`, `ReadPlan` are process-agnostic (they just check files/sockets over SSH). But `ReadRelayOutput` moved to `Backend` since it needs backend-specific parsing.
- **Relay path constants exported** — `agent.RelayDir`, `agent.RelayScriptPath`, `agent.RelaySockPath`, `agent.RelayOutputPath` are now exported so `claude/` can reference them.

### Current file layout

```
agent/
  agent.go       — Session (writeFn-pluggable), Options, ParseMessage, readMessages, relay utils
  backend.go     — Backend interface
  types.go       — Message interface + all message types (unchanged)
  agent_test.go  — Session/parse tests (updated for writeFn)
  claude/
    claude.go       — Backend impl: Start (relay deploy + serve-attach), AttachRelay, ReadRelayOutput, ParseMessage, WritePrompt, buildArgs, slogWriter
    claude_test.go  — WritePrompt test
  relay/
    embed.go     — relay.py embed (unchanged)
    relay.py     — unchanged
```

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

## Remaining Implementation Steps

### 2. Add `agent` field to task creation DTO — plumb through server/runner

- Add `Agent string` to `dto.CreateTaskReq` (default `"claude"`)
- Store agent name on `task.Task`
- `server.Server` holds a `map[string]agent.Backend` registry (currently just `{"claude": &claude.Backend{}}`)
- Runner uses the backend from the registry for the task's agent
- `task/load.go` — `agent.ParseMessage` in `loadLogFile` currently uses the package-level function. For multi-backend log replay, the log needs to record which backend produced it (add `Agent` field to `MetaMessage`), and `LoadLogs` must accept a backend registry or parse function map.

### 3. Implement Gemini backend — `agent/gemini/`

Create `agent/gemini/gemini.go` implementing `agent.Backend`:
- `Start`: launches `gemini -p --output-format stream-json --yolo [-m model]`
- `WritePrompt`: Gemini's stdin format (needs empirical capture — may differ from Claude's)
- `ParseMessage`: translates Gemini stream-json events → `agent.Message`
- Tool name mapping: `read_file`→`Read`, `run_shell_command`→`Bash`, etc.

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

### 4. Frontend: agent selector

- Task creation form gets an agent selector dropdown
- `EventInit` already has `Model` and `ClaudeCodeVersion`; rename `ClaudeCodeVersion` → `AgentVersion` in dto
- Display the agent name in the task header

### 5. Rename `ClaudeCodeVersion` → `AgentVersion` throughout dto/frontend

Touch points:
- `agent/types.go`: `SystemInitMessage.Version` JSON tag `claude_code_version`
- `server/dto/events.go`: `EventInit.ClaudeCodeVersion`
- `task/task.go`: `Task.ClaudeCodeVersion`, `addMessage`, `RestoreMessages`
- Frontend: wherever `ClaudeCodeVersion` appears in the TypeScript types and UI

## Risk Assessment

- **Gemini stream-json format is under-documented.** Need to empirically capture output from `gemini -p "hello" --output-format stream-json` and reverse-engineer the event schema. The format was stabilized in v0.20+ but exact field names need validation.
- **Relay compatibility.** The relay.py is process-agnostic (stdin/stdout bridge), so it should work with Gemini CLI unchanged. The only assumption is that the child process reads stdin and writes stdout — both CLIs do this.
- **Cost tracking.** Gemini has a free tier (no cost). The `ResultMessage.TotalCostUSD` would be 0 or derived from Gemini's stats if available.
- **Session resume.** Gemini's `--resume` works differently (local file-based sessions). Need to verify it works in a container where `~/.gemini/` persists.

## Non-Goals (for now)
- MCP server integration (both CLIs support it, but not needed for the core abstraction)
- Multiple simultaneous backends in a single task
- Backend-specific settings UI (model lists, API key management)
