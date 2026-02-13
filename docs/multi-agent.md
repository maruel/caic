# Multi-Agent Abstraction Plan for CAIC

## Completed: Backend Interface + Claude Subpackage (Step 1)

The `agent.Backend` interface is extracted and the Claude-specific code lives in `agent/claude/`.

- **`agent/backend.go`** — `Backend` interface with `Start`, `AttachRelay`, `ReadRelayOutput`, `ParseMessage`, `Harness`
- **`agent/claude/claude.go`** — Claude Code backend: launches `claude -p` via relay, parses Claude NDJSON, writes Claude stdin format
- **`agent/agent.go`** — Shared infrastructure: `Session` (with pluggable `WriteFn`), `ParseMessage`, `Options`, relay utilities
- **`WriteFn` on `Session`** instead of `WritePrompt` on `Backend` — avoids changing `task.Task.SendInput` or storing the backend on the task
- **Relay utilities stay in `agent/`** — `DeployRelay`, `IsRelayRunning`, `HasRelayDir`, `ReadPlan` are process-agnostic

## Completed: Harness Field + Backend Registry (Step 2)

The `harness` field is plumbed end-to-end from API request through task lifecycle, logs, and frontend display.

### What was done

- **`agent/types.go`** — `type Harness string` (defined type) with `agent.Claude` constant. `DiffFileStat`, `DiffStat` structs. `MetaMessage.Harness agent.Harness` (required, validated). The `agent` package has **no dependency on `dto`**.
- **`agent/backend.go`** — `Backend.Harness()` returns `agent.Harness`.
- **`dto/types.go`** — `type Harness string` (defined type). `HarnessClaude` constant. Duplicated `DiffFileStat`/`DiffStat` structs. Values must match `agent.Harness` constants.
- **`dto/validate.go`** — `CreateTaskReq.Validate()` requires non-empty `harness`.
- **`task/task.go`** — `Task.Harness agent.Harness`. No `dto` import.
- **`task/runner.go`** — `Runner.Backends map[agent.Harness]agent.Backend`. `runner.backend(name)` is a direct map lookup (no fallback). No `dto` import.
- **`task/diffstat.go`** — `ParseDiffNumstat` returns `agent.DiffStat`. No `dto` import.
- **`task/load.go`** — `LoadedTask.Harness agent.Harness`.
- **`server/server.go`** — Boundary conversions: `agent.Harness(req.Harness)` inbound, `dto.Harness(e.task.Harness)` outbound, `toDTODiffStat()` for DiffStat.
- **`server/eventconv.go`** — `toDTODiffStat()` conversion helper.
- **Frontend** — `createTask` passes `harness: "claude"`. `TaskItemSummary` shows harness name when not "claude".

### Key design decisions

- **`type Harness string`** (defined type in both `agent` and `dto`) — provides compile-time type safety. Named "Harness" because it identifies the CLI harness (Claude Code, Gemini CLI, opencode), not the model.
- **`agent.Harness` / `dto.Harness`** — both defined types, explicit conversions at the server boundary: `agent.Harness(req.Harness)` inbound, `dto.Harness(task.Harness)` outbound.
- **`agent.DiffFileStat` / `dto.DiffFileStat`** — duplicated struct types. `agent` owns the canonical definition; `dto` owns the API/frontend definition. Conversion at server boundary.
- **Required everywhere, no fallback** — `harness` field is mandatory in API requests and JSONL logs. `runner.backend()` does a direct map lookup.
- **`Backends` map on `Runner`** (not on `Server`) — keeps backend selection close to agent launch code.
- **`ParseMessage` in `loadLogFile` still uses `agent.ParseMessage`** (Claude format). When a new harness is added, `loadLogFile` will need per-harness dispatch. Deferred.

### Current file layout

```
agent/
  agent.go       — Session (writeFn-pluggable), Options, ParseMessage, readMessages, relay utils
  backend.go     — Backend interface (Harness() returns Harness)
  types.go       — Harness/Claude, DiffFileStat/DiffStat, Message types
  agent_test.go  — Session/parse tests
  claude/
    claude.go       — Backend impl (Harness() → agent.Claude)
    claude_test.go
  relay/
    embed.go     — relay.py embed
    relay.py

task/
  task.go        — Task struct (Harness agent.Harness); no dto import
  runner.go      — Runner with Backends map[agent.Harness]agent.Backend; no dto import
  diffstat.go    — ParseDiffNumstat → agent.DiffStat; no dto import
  load.go        — LoadedTask (Harness agent.Harness), loadLogFile

server/
  server.go      — agent.Harness↔dto.Harness + toDTODiffStat() boundary conversions
  eventconv.go   — agent.Message → dto.EventMessage + toDTODiffStat()

server/dto/
  types.go       — type Harness string, HarnessClaude, DiffFileStat/DiffStat (duplicated)
  validate.go    — CreateTaskReq.Validate() (harness required)
  events.go      — EventInit (still has ClaudeCodeVersion — rename is step 5)
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

### 3. Implement Gemini backend — `agent/gemini/`

Create `agent/gemini/gemini.go` implementing `agent.Backend`:
- `Start`: launches `gemini -p --output-format stream-json --yolo [-m model]`
- `WritePrompt`: Gemini's stdin format (needs empirical capture — may differ from Claude's)
- `ParseMessage`: translates Gemini stream-json events → `agent.Message`
- Tool name mapping: `read_file`→`Read`, `run_shell_command`→`Bash`, etc.
- Register in `runner.initDefaults` as `{"gemini": &gemini.Backend{}}` alongside claude

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

**Important for the implementer:** Once a Gemini backend exists, `task/load.go:loadLogFile` needs updating — currently it calls `agent.ParseMessage` (Claude format) for all log lines. It should read `meta.Harness` and dispatch to the correct backend's `ParseMessage`. The `LoadLogs` function (or `loadLogFile`) will need a `map[string]agent.Backend` parameter or a parse-function registry.

### 4. Frontend: harness selector

- Task creation form gets a harness selector dropdown (populate from available backends — may need a new `/api/v1/harnesses` endpoint or include in repos response)
- Display the harness name in the task header (already shown in `TaskItemSummary` for non-claude harnesses)

### 5. Rename `ClaudeCodeVersion` → `AgentVersion` throughout dto/frontend

Touch points:
- `agent/types.go`: `SystemInitMessage.Version` JSON tag `claude_code_version` — Gemini will populate this with its own version
- `server/dto/events.go`: `EventInit.ClaudeCodeVersion` → `AgentVersion`
- `task/task.go`: `Task.ClaudeCodeVersion` → `AgentVersion`, `addMessage`, `RestoreMessages`
- `server/dto/types.go`: `TaskJSON.ClaudeCodeVersion` → `AgentVersion`
- `server/server.go`: `toJSON` mapping
- `server/eventconv.go`: `convertMessage` init case
- Frontend: `TaskItemSummary.tsx` (props + display), `TaskList.tsx` (prop pass-through), `TaskView.tsx` (session started display)
- Tests: `server_test.go`, `eventconv_test.go`, `agent_test.go` — update field names and JSON literals

## Risk Assessment

- **Gemini stream-json format is under-documented.** Need to empirically capture output from `gemini -p "hello" --output-format stream-json` and reverse-engineer the event schema. The format was stabilized in v0.20+ but exact field names need validation.
- **Relay compatibility.** The relay.py is process-agnostic (stdin/stdout bridge), so it should work with Gemini CLI unchanged. The only assumption is that the child process reads stdin and writes stdout — both CLIs do this.
- **Cost tracking.** Gemini has a free tier (no cost). The `ResultMessage.TotalCostUSD` would be 0 or derived from Gemini's stats if available.
- **Session resume.** Gemini's `--resume` works differently (local file-based sessions). Need to verify it works in a container where `~/.gemini/` persists.

## Non-Goals (for now)
- MCP server integration (both CLIs support it, but not needed for the core abstraction)
- Multiple simultaneous backends in a single task
- Backend-specific settings UI (model lists, API key management)
