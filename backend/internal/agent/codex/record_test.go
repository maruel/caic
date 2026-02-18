package codex

import (
	"encoding/json"
	"testing"
)

func TestRecord(t *testing.T) {
	t.Run("ThreadStarted", func(t *testing.T) {
		const input = `{"type":"thread.started","thread_id":"0199a213-81c0-7800-8aa1-bbab2a035a53"}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		if rec.Type != TypeThreadStarted {
			t.Fatalf("Type = %q, want %q", rec.Type, TypeThreadStarted)
		}
		r, err := rec.AsThreadStarted()
		if err != nil {
			t.Fatal(err)
		}
		if r.ThreadID != "0199a213-81c0-7800-8aa1-bbab2a035a53" {
			t.Errorf("ThreadID = %q", r.ThreadID)
		}
		if len(r.Extra) != 0 {
			t.Errorf("unexpected extra fields: %v", r.Extra)
		}
	})
	t.Run("UnknownFields", func(t *testing.T) {
		const input = `{"type":"thread.started","thread_id":"t1","new_field":"surprise"}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsThreadStarted()
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Extra) != 1 {
			t.Fatalf("Extra = %v, want 1 unknown field", r.Extra)
		}
		if _, ok := r.Extra["new_field"]; !ok {
			t.Error("expected 'new_field' in Extra")
		}
	})
	t.Run("TurnCompleted", func(t *testing.T) {
		const input = `{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122}}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		if rec.Type != TypeTurnCompleted {
			t.Fatalf("Type = %q, want %q", rec.Type, TypeTurnCompleted)
		}
		r, err := rec.AsTurnCompleted()
		if err != nil {
			t.Fatal(err)
		}
		if r.Usage.InputTokens != 24763 {
			t.Errorf("InputTokens = %d", r.Usage.InputTokens)
		}
		if r.Usage.CachedInputTokens != 24448 {
			t.Errorf("CachedInputTokens = %d", r.Usage.CachedInputTokens)
		}
		if r.Usage.OutputTokens != 122 {
			t.Errorf("OutputTokens = %d", r.Usage.OutputTokens)
		}
	})
	t.Run("TurnFailed", func(t *testing.T) {
		const input = `{"type":"turn.failed","error":"something went wrong"}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsTurnFailed()
		if err != nil {
			t.Fatal(err)
		}
		if r.Error != "something went wrong" {
			t.Errorf("Error = %q", r.Error)
		}
	})
	t.Run("ItemCommandExecution", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc ls","aggregated_output":"docs\nsrc\n","exit_code":0,"status":"completed"}}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsItem()
		if err != nil {
			t.Fatal(err)
		}
		if r.Item.ID != "item_1" {
			t.Errorf("ID = %q", r.Item.ID)
		}
		if r.Item.Type != ItemTypeCommandExecution {
			t.Errorf("Type = %q", r.Item.Type)
		}
		if r.Item.Command != "bash -lc ls" {
			t.Errorf("Command = %q", r.Item.Command)
		}
		if r.Item.AggregatedOutput != "docs\nsrc\n" {
			t.Errorf("AggregatedOutput = %q", r.Item.AggregatedOutput)
		}
		if r.Item.ExitCode == nil || *r.Item.ExitCode != 0 {
			t.Errorf("ExitCode = %v", r.Item.ExitCode)
		}
		if r.Item.Status != "completed" {
			t.Errorf("Status = %q", r.Item.Status)
		}
	})
	t.Run("ItemFileChange", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_4","type":"file_change","changes":[{"path":"docs/foo.md","kind":"add"}],"status":"completed"}}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsItem()
		if err != nil {
			t.Fatal(err)
		}
		if r.Item.Type != ItemTypeFileChange {
			t.Errorf("Type = %q", r.Item.Type)
		}
		if len(r.Item.Changes) != 1 {
			t.Fatalf("Changes = %d, want 1", len(r.Item.Changes))
		}
		if r.Item.Changes[0].Path != "docs/foo.md" {
			t.Errorf("Path = %q", r.Item.Changes[0].Path)
		}
		if r.Item.Changes[0].Kind != "add" {
			t.Errorf("Kind = %q", r.Item.Changes[0].Kind)
		}
	})
	t.Run("ItemAgentMessage", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Done.","status":"completed"}}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsItem()
		if err != nil {
			t.Fatal(err)
		}
		if r.Item.Type != ItemTypeAgentMessage {
			t.Errorf("Type = %q", r.Item.Type)
		}
		if r.Item.Text != "Done." {
			t.Errorf("Text = %q", r.Item.Text)
		}
	})
	t.Run("ItemReasoning", func(t *testing.T) {
		const input = `{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"**Scanning...**","status":"completed"}}`
		var rec Record
		if err := json.Unmarshal([]byte(input), &rec); err != nil {
			t.Fatal(err)
		}
		r, err := rec.AsItem()
		if err != nil {
			t.Fatal(err)
		}
		if r.Item.Type != ItemTypeReasoning {
			t.Errorf("Type = %q", r.Item.Type)
		}
		if r.Item.Text != "**Scanning...**" {
			t.Errorf("Text = %q", r.Item.Text)
		}
	})
}
