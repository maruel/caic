// Tests for groupMessages and groupTurns logic.
import { describe, it, expect } from "vitest";
import { groupMessages, groupTurns, turnSummary } from "./grouping";
import type { EventMessage } from "@sdk/types.gen";

function toolUseEvent(id: string, name: string): EventMessage {
  return { kind: "toolUse", ts: 0, toolUse: { toolUseID: id, name, input: {} } };
}

function toolResultEvent(id: string): EventMessage {
  return { kind: "toolResult", ts: 0, toolResult: { toolUseID: id, duration: 0.1 } };
}

function textDeltaEvent(text: string): EventMessage {
  return { kind: "textDelta", ts: 0, textDelta: { text } };
}

function usageEvent(): EventMessage {
  return {
    kind: "usage", ts: 0,
    usage: { inputTokens: 100, outputTokens: 50, cacheCreationInputTokens: 0, cacheReadInputTokens: 0, model: "test" },
  };
}

function resultEvent(): EventMessage {
  return {
    kind: "result", ts: 0,
    result: {
      subtype: "success", isError: false, result: "done",
      totalCostUSD: 0.01, duration: 1.0, durationAPI: 0.9,
      numTurns: 1,
      usage: { inputTokens: 100, outputTokens: 50, cacheCreationInputTokens: 0, cacheReadInputTokens: 0, model: "test" },
    },
  };
}

describe("groupMessages", () => {
  it("consecutive tool uses form one group", () => {
    const groups = groupMessages([toolUseEvent("t1", "Read"), toolUseEvent("t2", "Bash")]);
    expect(groups).toHaveLength(1);
    expect(groups[0].kind).toBe("action");
    expect(groups[0].toolCalls).toHaveLength(2);
  });

  it("synchronous tools in last group are done immediately", () => {
    // Bash is async (emits toolResult); Read is synchronous (no toolResult).
    // Even before Bash's result arrives, Read should show as done.
    const groups = groupMessages([toolUseEvent("t1", "Bash"), toolUseEvent("t2", "Read")]);
    expect(groups).toHaveLength(1);
    expect(groups[0].toolCalls[0].done).toBe(false); // Bash: async, pending
    expect(groups[0].toolCalls[1].done).toBe(true);  // Read: sync, already done
  });

  it("async tool is marked done when its toolResult arrives", () => {
    const groups = groupMessages([
      toolUseEvent("t1", "Bash"),
      toolResultEvent("t1"),
    ]);
    expect(groups[0].toolCalls[0].done).toBe(true);
    expect(groups[0].toolCalls[0].result?.toolUseID).toBe("t1");
  });

  it("toolResult matches backwards across groups", () => {
    const groups = groupMessages([
      toolUseEvent("t1", "Bash"),
      textDeltaEvent("text"),
      toolResultEvent("t1"),
    ]);
    expect(groups).toHaveLength(2);
    expect(groups[0].kind).toBe("action");
    expect(groups[0].toolCalls[0].done).toBe(true);
    expect(groups[0].toolCalls[0].result?.toolUseID).toBe("t1");
  });

  it("non-last tool groups are implicitly marked done", () => {
    const groups = groupMessages([
      toolUseEvent("t1", "Read"),
      usageEvent(),
      textDeltaEvent("text"),
      toolUseEvent("t2", "Bash"),
    ]);
    // After merge pass: [TOOL(Read, Bash), TEXT]
    expect(groups).toHaveLength(2);
    expect(groups[0].toolCalls[0].done).toBe(true); // Read
    expect(groups[0].toolCalls[1].done).toBe(true); // Bash (implicit from non-last-group pass)
  });

  it("tool groups separated by text merge into one", () => {
    const groups = groupMessages([
      toolUseEvent("t1", "Read"),
      usageEvent(),
      textDeltaEvent("commentary"),
      toolUseEvent("t2", "Bash"),
      usageEvent(),
      textDeltaEvent("more"),
      toolUseEvent("t3", "Edit"),
    ]);
    expect(groups).toHaveLength(3); // [TOOL(t1+t2+t3), TEXT, TEXT]
    expect(groups[0].kind).toBe("action");
    expect(groups[0].toolCalls).toHaveLength(3);
  });

  it("thinking events are absorbed into an adjacent tool group", () => {
    // Realistic pattern: usage ends the first assistant message, then thinking
    // precedes the next tool call in a new assistant message.
    const groups = groupMessages([
      toolUseEvent("t1", "Read"),
      usageEvent(),
      { kind: "thinking", ts: 0, thinking: { text: "hmm" } },
      { kind: "subagentStart", ts: 0, subagentStart: { taskID: "sa1", description: "explore" } },
      toolUseEvent("t2", "Bash"),
      { kind: "subagentEnd", ts: 0, subagentEnd: { taskID: "sa1", status: "completed" } },
    ]);
    // Thinking is absorbed into the merged tool group; no standalone thinking group.
    const toolGroup = groups.find((g) => g.kind === "action");
    expect(toolGroup?.toolCalls).toHaveLength(2);
    expect(toolGroup?.events.some((e) => e.kind === "thinking")).toBe(true);
    // Subagent events don't create groups.
    expect(groups.some((g) => g.kind === "other")).toBe(false);
  });

  it("thinking followed by usage does not create a barrier before tool use", () => {
    // usage after a thinking-only group must not create an OTHER barrier that
    // prevents the merge pass from absorbing thinking into the tool group.
    const groups = groupMessages([
      { kind: "thinkingDelta", ts: 0, thinkingDelta: { text: "thinking..." } },
      usageEvent(),
      toolUseEvent("t1", "Read"),
    ]);
    expect(groups.some((g) => g.kind === "other")).toBe(false);
    expect(groups).toHaveLength(1);
    const toolGroup = groups[0];
    expect(toolGroup.kind).toBe("action");
    expect(toolGroup.toolCalls).toHaveLength(1);
    expect(toolGroup.events.some((e) => e.kind === "thinkingDelta")).toBe(true);
  });

  it("thinking immediately after a tool group is absorbed into it", () => {
    // The agent may start a new thinking block right after tool calls complete,
    // before any text commentary. It should merge into the preceding tool group.
    const groups = groupMessages([
      toolUseEvent("t1", "Read"),
      usageEvent(),
      { kind: "thinkingDelta", ts: 0, thinkingDelta: { text: "analyzing..." } },
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0].kind).toBe("action");
    expect(groups[0].toolCalls).toHaveLength(1);
    expect(groups[0].events.some((e) => e.kind === "thinkingDelta")).toBe(true);
  });

  it("thinking followed by text is absorbed into the text group", () => {
    // Standalone thinking before text commentary (no tools) must not produce a
    // separate Thinking block; it should be inside the text group instead.
    const groups = groupMessages([
      { kind: "thinkingDelta", ts: 0, thinkingDelta: { text: "thinking..." } },
      textDeltaEvent("hello"),
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0].kind).toBe("text");
    expect(groups[0].events.some((e) => e.kind === "thinkingDelta")).toBe(true);
    expect(groups[0].events.some((e) => e.kind === "textDelta")).toBe(true);
  });
});

describe("groupTurns", () => {
  it("result event splits turns", () => {
    const events: EventMessage[] = [
      textDeltaEvent("first turn"),
      toolUseEvent("t1", "Read"),
      resultEvent(),
      textDeltaEvent("second turn"),
    ];
    const groups = groupMessages(events);
    const turns = groupTurns(groups);
    expect(turns).toHaveLength(2);
    expect(turns[0].toolCount).toBe(1);
    expect(turns[0].textCount).toBe(1);
    expect(turns[1].toolCount).toBe(0);
    expect(turns[1].textCount).toBe(1);
  });

  it("turnSummary formats correctly", () => {
    const turn = { groups: [], toolCount: 3, textCount: 2 };
    expect(turnSummary(turn)).toBe("2 messages, 3 tool calls");
  });
});
