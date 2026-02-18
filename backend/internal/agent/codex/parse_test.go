package codex

import (
	"testing"

	"github.com/maruel/caic/backend/internal/agent"
)

func TestParseMessage(t *testing.T) {
	t.Run("ThreadStarted", func(t *testing.T) {
		const input = `{"type":"thread.started","thread_id":"0199a213-81c0-7800-8aa1-bbab2a035a53"}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		init, ok := msg.(*agent.SystemInitMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.SystemInitMessage", msg)
		}
		if init.SessionID != "0199a213-81c0-7800-8aa1-bbab2a035a53" {
			t.Errorf("SessionID = %q", init.SessionID)
		}
	})
	t.Run("TurnStarted", func(t *testing.T) {
		const input = `{"type":"turn.started"}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		sm, ok := msg.(*agent.SystemMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.SystemMessage", msg)
		}
		if sm.Subtype != "turn_started" {
			t.Errorf("Subtype = %q", sm.Subtype)
		}
	})
	t.Run("TurnCompleted", func(t *testing.T) {
		const input = `{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		rm, ok := msg.(*agent.ResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ResultMessage", msg)
		}
		if rm.IsError {
			t.Error("IsError should be false")
		}
		if rm.Usage.InputTokens != 24763 {
			t.Errorf("InputTokens = %d", rm.Usage.InputTokens)
		}
		if rm.Usage.OutputTokens != 122 {
			t.Errorf("OutputTokens = %d", rm.Usage.OutputTokens)
		}
		if rm.Usage.CacheReadInputTokens != 24448 {
			t.Errorf("CacheReadInputTokens = %d", rm.Usage.CacheReadInputTokens)
		}
	})
	t.Run("TurnFailed", func(t *testing.T) {
		const input = `{"type":"turn.failed","error":"rate limit exceeded"}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		rm, ok := msg.(*agent.ResultMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.ResultMessage", msg)
		}
		if !rm.IsError {
			t.Error("IsError should be true")
		}
		if rm.Result != "rate limit exceeded" {
			t.Errorf("Result = %q", rm.Result)
		}
	})
	t.Run("ItemStartedCommandExecution", func(t *testing.T) {
		const input = `{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"","exit_code":null,"status":"in_progress"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		if len(am.Message.Content) != 1 {
			t.Fatalf("content blocks = %d, want 1", len(am.Message.Content))
		}
		cb := am.Message.Content[0]
		if cb.Type != "tool_use" {
			t.Errorf("content type = %q, want tool_use", cb.Type)
		}
		if cb.Name != "Bash" {
			t.Errorf("Name = %q, want Bash", cb.Name)
		}
		if cb.ID != "item_1" {
			t.Errorf("ID = %q", cb.ID)
		}
	})
	t.Run("ItemCompletedCommandExecution", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"docs\nsrc\n","exit_code":0,"status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		um, ok := msg.(*agent.UserMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.UserMessage", msg)
		}
		if um.ParentToolUseID == nil || *um.ParentToolUseID != "item_1" {
			t.Errorf("ParentToolUseID = %v", um.ParentToolUseID)
		}
	})
	t.Run("ItemCompletedAgentMessage", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Done.","status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		if len(am.Message.Content) != 1 {
			t.Fatalf("content blocks = %d, want 1", len(am.Message.Content))
		}
		if am.Message.Content[0].Text != "Done." {
			t.Errorf("text = %q", am.Message.Content[0].Text)
		}
	})
	t.Run("ItemCompletedReasoning", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"**Scanning...**","status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		if am.Message.Content[0].Type != "text" {
			t.Errorf("content type = %q", am.Message.Content[0].Type)
		}
		if am.Message.Content[0].Text != "**Scanning...**" {
			t.Errorf("text = %q", am.Message.Content[0].Text)
		}
	})
	t.Run("ItemCompletedFileChangeAdd", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_4","type":"file_change","changes":[{"path":"docs/foo.md","kind":"add"}],"status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		cb := am.Message.Content[0]
		if cb.Name != "Write" {
			t.Errorf("Name = %q, want Write (file add)", cb.Name)
		}
	})
	t.Run("ItemCompletedFileChangeUpdate", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_5","type":"file_change","changes":[{"path":"src/main.go","kind":"update"}],"status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		cb := am.Message.Content[0]
		if cb.Name != "Edit" {
			t.Errorf("Name = %q, want Edit (file update)", cb.Name)
		}
	})
	t.Run("ItemCompletedWebSearch", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_6","type":"web_search","query":"golang generics","status":"completed"}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		am, ok := msg.(*agent.AssistantMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.AssistantMessage", msg)
		}
		cb := am.Message.Content[0]
		if cb.Name != "WebSearch" {
			t.Errorf("Name = %q, want WebSearch", cb.Name)
		}
	})
	t.Run("ItemUpdated", func(t *testing.T) {
		const input = `{"type":"item.updated","item":{"id":"item_1","type":"command_execution","aggregated_output":"partial..."}}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		raw, ok := msg.(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msg)
		}
		if raw.Type() != TypeItemUpdated {
			t.Errorf("Type() = %q", raw.Type())
		}
	})
	t.Run("Error", func(t *testing.T) {
		const input = `{"type":"error","message":"non-fatal warning"}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		raw, ok := msg.(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msg)
		}
		if raw.Type() != TypeError {
			t.Errorf("Type() = %q", raw.Type())
		}
	})
	t.Run("UnknownType", func(t *testing.T) {
		const input = `{"type":"future_event","data":"something"}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		raw, ok := msg.(*agent.RawMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.RawMessage", msg)
		}
		if raw.Type() != "future_event" {
			t.Errorf("Type() = %q", raw.Type())
		}
	})
	t.Run("DiffStat", func(t *testing.T) {
		const input = `{"type":"caic_diff_stat","diff_stat":[{"path":"foo.go","added":10,"deleted":2}]}`
		msg, err := ParseMessage([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		ds, ok := msg.(*agent.DiffStatMessage)
		if !ok {
			t.Fatalf("type = %T, want *agent.DiffStatMessage", msg)
		}
		if len(ds.DiffStat) != 1 {
			t.Fatalf("DiffStat = %d, want 1", len(ds.DiffStat))
		}
		if ds.DiffStat[0].Path != "foo.go" {
			t.Errorf("Path = %q", ds.DiffStat[0].Path)
		}
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("WithPrompt", func(t *testing.T) {
		args := buildArgs(&agent.Options{Prompt: "fix the bug"})
		last := args[len(args)-1]
		if last != "fix the bug" {
			t.Errorf("last arg = %q, want prompt", last)
		}
	})
	t.Run("WithModel", func(t *testing.T) {
		args := buildArgs(&agent.Options{Model: "o4-mini", Prompt: "hello"})
		// Model flag should appear before the trailing prompt.
		found := false
		for i, a := range args {
			if a == "-m" && i+1 < len(args) && args[i+1] == "o4-mini" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("args = %v, missing -m o4-mini", args)
		}
		if args[len(args)-1] != "hello" {
			t.Errorf("last arg = %q, want prompt", args[len(args)-1])
		}
	})
	t.Run("NoPrompt", func(t *testing.T) {
		args := buildArgs(&agent.Options{})
		last := args[len(args)-1]
		if last != "--full-auto" {
			t.Errorf("last arg = %q, want --full-auto (no trailing prompt)", last)
		}
	})
}

func TestParseMessageFullStream(t *testing.T) {
	// Parse the full example stream from the spec.
	lines := []string{
		`{"type":"thread.started","thread_id":"0199a213-81c0-7800-8aa1-bbab2a035a53"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"**Scanning...**","status":"completed"}}`,
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"","exit_code":null,"status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"docs\nsrc\n","exit_code":0,"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_4","type":"file_change","changes":[{"path":"docs/foo.md","kind":"add"}],"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Done.","status":"completed"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122}}`,
	}
	wantTypes := []string{
		"system",    // thread.started → SystemInitMessage
		"system",    // turn.started → SystemMessage
		"assistant", // reasoning → AssistantMessage
		"assistant", // item.started command_execution → AssistantMessage (tool_use)
		"user",      // item.completed command_execution → UserMessage (tool result)
		"assistant", // file_change → AssistantMessage (tool_use)
		"assistant", // agent_message → AssistantMessage
		"result",    // turn.completed → ResultMessage
	}
	for i, line := range lines {
		msg, err := ParseMessage([]byte(line))
		if err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		if msg.Type() != wantTypes[i] {
			t.Errorf("line %d: Type() = %q, want %q", i, msg.Type(), wantTypes[i])
		}
	}
}
