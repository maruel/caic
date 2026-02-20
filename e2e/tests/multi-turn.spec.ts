// Multi-turn interaction and concurrent task tests.
import { test, expect, createTaskAPI, waitForTaskState } from "../helpers";

test("multi-turn: send input cycles to next joke", async ({ page, api }) => {
  const id = await createTaskAPI(api, "multi-turn test");
  await waitForTaskState(api, id, "waiting");

  // Open the task in the browser to verify streaming output.
  await page.goto("/");
  await page.getByText("multi-turn test").first().click();

  // First joke should be visible.
  await expect(
    page.getByText("Why do programmers prefer dark mode?").first(),
  ).toBeVisible({ timeout: 15_000 });

  // Send input to trigger the second turn.
  await api.sendInput(id, { prompt: { text: "tell me another" } });
  await waitForTaskState(api, id, "waiting", 20_000);

  // Second joke: "A SQL query walks into a bar..."
  await expect(
    page.getByText("A SQL query walks into a bar").first(),
  ).toBeVisible({ timeout: 15_000 });
});

test("concurrent tasks run independently", async ({ api }) => {
  const id1 = await createTaskAPI(api, "concurrent task A");
  const id2 = await createTaskAPI(api, "concurrent task B");

  // Both should reach waiting state independently.
  const [task1, task2] = await Promise.all([
    waitForTaskState(api, id1, "waiting"),
    waitForTaskState(api, id2, "waiting"),
  ]);

  expect(task1.id).not.toBe(task2.id);
  expect(task1.branch).not.toBe(task2.branch);

  // Terminate both.
  await Promise.all([
    api.terminateTask(id1),
    api.terminateTask(id2),
  ]);
  await Promise.all([
    waitForTaskState(api, id1, "terminated"),
    waitForTaskState(api, id2, "terminated"),
  ]);
});

test("SSE event stream delivers text deltas", async ({ page, api, baseURL }) => {
  const id = await createTaskAPI(api, "sse stream test");

  // Navigate to the app first so the browser has the right origin.
  await page.goto("/");

  // Consume SSE events in the browser context using the full URL.
  const events = await page.evaluate(async ({ taskId, base }) => {
    return new Promise<string[]>((resolve) => {
      const collected: string[] = [];
      const es = new EventSource(`${base}/api/v1/tasks/${taskId}/events`);
      es.addEventListener("message", (e) => {
        const msg = JSON.parse(e.data);
        collected.push(msg.kind);
        if (msg.kind === "result") {
          es.close();
          resolve(collected);
        }
      });
      // Safety timeout.
      setTimeout(() => { es.close(); resolve(collected); }, 15_000);
    });
  }, { taskId: id, base: baseURL! });

  // The fake agent should emit: init, textDelta(s), text, result.
  expect(events).toContain("init");
  expect(events).toContain("textDelta");
  expect(events).toContain("text");
  expect(events).toContain("result");
});
