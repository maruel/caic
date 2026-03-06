package kilo

import (
	"testing"

	"github.com/caic-xyz/caic/backend/internal/agent"
)

func TestParseMessage(t *testing.T) {
	t.Run("Init", func(t *testing.T) {
		const input = `{"type":"system","subtype":"init","session_id":"ses_abc","model":"anthropic/claude-sonnet-4-20250514"}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		init, ok := msgs[0].(*agent.InitMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.InitMessage", msgs[0])
		}
		if init.SessionID != "ses_abc" {
			t.Errorf("SessionID = %q", init.SessionID)
		}
		if init.Model != "anthropic/claude-sonnet-4-20250514" {
			t.Errorf("Model = %q", init.Model)
		}
	})

	t.Run("TextCompleted", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_1","sessionID":"ses_abc","messageID":"msg_1","type":"text","text":"Hello world","time":{"start":1234567889000,"end":1234567890000}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		tm, ok := msgs[0].(*agent.TextMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.TextMessage", msgs[0])
		}
		if tm.Text != "Hello world" {
			t.Errorf("Text = %q", tm.Text)
		}
	})

	t.Run("ToolRunning", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_2","sessionID":"ses_abc","messageID":"msg_1","type":"tool","callID":"call_1","tool":"bash","state":{"status":"running","input":{"command":"ls"}}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		tu, ok := msgs[0].(*agent.ToolUseMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ToolUseMessage", msgs[0])
		}
		if tu.Name != "Bash" {
			t.Errorf("Name = %q, want Bash (normalized from bash)", tu.Name)
		}
		if tu.ToolUseID != "call_1" {
			t.Errorf("ToolUseID = %q", tu.ToolUseID)
		}
	})

	t.Run("ToolCompleted", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_2","sessionID":"ses_abc","messageID":"msg_1","type":"tool","callID":"call_1","tool":"bash","state":{"status":"completed","input":{"command":"ls"},"output":"file1.txt\nfile2.txt"}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		tr, ok := msgs[0].(*agent.ToolResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ToolResultMessage", msgs[0])
		}
		if tr.ToolUseID != "call_1" {
			t.Errorf("ToolUseID = %q", tr.ToolUseID)
		}
		if tr.Error != "" {
			t.Errorf("Error = %q, want empty", tr.Error)
		}
	})

	t.Run("ToolError", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_3","sessionID":"ses_abc","messageID":"msg_1","type":"tool","callID":"call_2","tool":"read","state":{"status":"error","input":{"file_path":"/etc/shadow"},"error":"Permission denied"}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		tr, ok := msgs[0].(*agent.ToolResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ToolResultMessage", msgs[0])
		}
		if tr.ToolUseID != "call_2" {
			t.Errorf("ToolUseID = %q", tr.ToolUseID)
		}
		if tr.Error != "Permission denied" {
			t.Errorf("Error = %q, want %q", tr.Error, "Permission denied")
		}
	})

	t.Run("StepFinish", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_4","sessionID":"ses_abc","messageID":"msg_1","type":"step-finish","cost":0.003,"tokens":{"total":1500,"input":500,"output":1000,"reasoning":0,"cache":{"read":100,"write":50}}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		rm, ok := msgs[0].(*agent.ResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ResultMessage", msgs[0])
		}
		if rm.IsError {
			t.Error("IsError should be false")
		}
		if rm.TotalCostUSD != 0.003 {
			t.Errorf("TotalCostUSD = %v, want 0.003", rm.TotalCostUSD)
		}
		if rm.Usage.InputTokens != 500 {
			t.Errorf("InputTokens = %d", rm.Usage.InputTokens)
		}
		if rm.Usage.OutputTokens != 1000 {
			t.Errorf("OutputTokens = %d", rm.Usage.OutputTokens)
		}
		if rm.Usage.CacheReadInputTokens != 100 {
			t.Errorf("CacheReadInputTokens = %d", rm.Usage.CacheReadInputTokens)
		}
		if rm.Usage.CacheCreationInputTokens != 50 {
			t.Errorf("CacheCreationInputTokens = %d", rm.Usage.CacheCreationInputTokens)
		}
	})

	t.Run("Reasoning", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_5","sessionID":"ses_abc","messageID":"msg_1","type":"reasoning","text":"Let me think...","time":{"start":1234567889000,"end":1234567890000}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		tm, ok := msgs[0].(*agent.ThinkingMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ThinkingMessage", msgs[0])
		}
		if tm.Text != "Let me think..." {
			t.Errorf("Text = %q", tm.Text)
		}
	})

	t.Run("StepStart", func(t *testing.T) {
		const input = `{"type":"message.part.updated","properties":{"part":{"id":"prt_6","sessionID":"ses_abc","messageID":"msg_1","type":"step-start","snapshot":"..."}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		sm, ok := msgs[0].(*agent.SystemMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.SystemMessage", msgs[0])
		}
		if sm.Subtype != "step_start" {
			t.Errorf("Subtype = %q, want step_start", sm.Subtype)
		}
	})

	t.Run("Delta", func(t *testing.T) {
		const input = `{"type":"message.part.delta","properties":{"sessionID":"ses_abc","messageID":"msg_1","partID":"prt_1","field":"text","delta":"Hello"}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		td, ok := msgs[0].(*agent.TextDeltaMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.TextDeltaMessage", msgs[0])
		}
		if td.Text != "Hello" {
			t.Errorf("Text = %q", td.Text)
		}
	})

	t.Run("SessionError", func(t *testing.T) {
		const input = `{"type":"session.error","properties":{"sessionID":"ses_abc","error":{"name":"UnknownError","data":{"message":"Model not found: google/gemini-3.1-flash-lite-preview."}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		rm, ok := msgs[0].(*agent.ResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ResultMessage", msgs[0])
		}
		if !rm.IsError {
			t.Error("IsError should be true")
		}
		if rm.Result != "Model not found: google/gemini-3.1-flash-lite-preview." {
			t.Errorf("Result = %q", rm.Result)
		}
	})

	t.Run("SessionErrorNoMessage", func(t *testing.T) {
		const input = `{"type":"session.error","properties":{"sessionID":"ses_abc","error":{"name":"ProviderAuthError","data":{}}}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		rm, ok := msgs[0].(*agent.ResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ResultMessage", msgs[0])
		}
		if rm.Result != "ProviderAuthError" {
			t.Errorf("Result = %q, want ProviderAuthError", rm.Result)
		}
	})

	t.Run("TurnCloseError", func(t *testing.T) {
		// Error details come from session.error; turn.close is passed through as raw.
		const input = `{"type":"session.turn.close","properties":{"sessionID":"ses_abc","reason":"error"}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		raw, ok := msgs[0].(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msgs[0])
		}
		if raw.Type() != TypeTurnClose {
			t.Errorf("Type() = %q", raw.Type())
		}
	})

	t.Run("TurnCloseCompleted", func(t *testing.T) {
		const input = `{"type":"session.turn.close","properties":{"sessionID":"ses_abc","reason":"completed"}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		raw, ok := msgs[0].(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msgs[0])
		}
		if raw.Type() != TypeTurnClose {
			t.Errorf("Type() = %q", raw.Type())
		}
	})

	t.Run("DiffStat", func(t *testing.T) {
		const input = `{"type":"caic_diff_stat","diff_stat":[{"path":"main.go","added":10,"deleted":2}]}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		ds, ok := msgs[0].(*agent.DiffStatMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.DiffStatMessage", msgs[0])
		}
		if len(ds.DiffStat) != 1 {
			t.Fatalf("DiffStat len = %d, want 1", len(ds.DiffStat))
		}
		if ds.DiffStat[0].Path != "main.go" {
			t.Errorf("Path = %q", ds.DiffStat[0].Path)
		}
	})

	t.Run("UnknownType", func(t *testing.T) {
		const input = `{"type":"session.turn.open","properties":{"sessionID":"ses_abc"}}`
		msgs, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("msgs = %d, want 1", len(msgs))
		}
		raw, ok := msgs[0].(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msgs[0])
		}
		if raw.Type() != "session.turn.open" {
			t.Errorf("Type() = %q", raw.Type())
		}
	})
}

func TestNormalizeToolName(t *testing.T) {
	tests := []struct {
		kilo string
		want string
	}{
		{"bash", "Bash"},
		{"read", "Read"},
		{"write", "Write"},
		{"edit", "Edit"},
		{"glob", "Glob"},
		{"grep", "Grep"},
		{"web_fetch", "WebFetch"},
		{"web_search", "WebSearch"},
		{"todo_write", "TodoWrite"},
		{"ask_user", "AskUserQuestion"},
		{"agent", "Agent"},
		{"some_new_tool", "some_new_tool"},
	}
	for _, tt := range tests {
		t.Run(tt.kilo, func(t *testing.T) {
			if got := normalizeToolName(tt.kilo); got != tt.want {
				t.Errorf("normalizeToolName(%q) = %q, want %q", tt.kilo, got, tt.want)
			}
		})
	}
}
