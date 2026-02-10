// End-to-end tests for the task lifecycle using a fake backend.
import { test, expect } from "@playwright/test";

test("create task, wait for result, and finish", async ({ page }) => {
  await page.goto("/");

  // Wait for repos to load (select gets an option).
  await expect(page.locator("select option")).not.toHaveCount(0);

  // Fill prompt and submit.
  await page.fill('input[placeholder="Describe a task..."]', "e2e test task");
  await page.getByRole("button", { name: "Run" }).click();

  // Click the task card to open TaskView.
  await page.getByText("e2e test task").first().click();

  // Wait for the assistant message from the fake agent.
  await expect(page.getByText("I completed the requested task.")).toBeVisible({
    timeout: 15_000,
  });

  // Wait for the result message.
  await expect(page.locator("strong", { hasText: "Done" })).toBeVisible({
    timeout: 10_000,
  });

  // The Finish button should appear once the task is in waiting state.
  const finishBtn = page.getByRole("button", { name: "Finish" });
  await expect(finishBtn).toBeVisible({ timeout: 10_000 });

  // Click Finish.
  await finishBtn.click();

  // Poll API until task state is "done".
  await expect(async () => {
    const resp = await page.request.get("/api/v1/tasks");
    const tasks = await resp.json();
    expect(tasks[0].state).toBe("done");
  }).toPass({ timeout: 15_000, intervals: [500] });
});

test("create task and end", async ({ page }) => {
  await page.goto("/");

  // Wait for repos.
  await expect(page.locator("select option")).not.toHaveCount(0);

  // Create task.
  await page.fill('input[placeholder="Describe a task..."]', "e2e end test");
  await page.getByRole("button", { name: "Run" }).click();

  // Open TaskView.
  await page.getByText("e2e end test").first().click();

  // Wait for the End button to be visible (task becomes active).
  const endBtn = page.getByRole("button", { name: "End", exact: true });
  await expect(endBtn).toBeVisible({ timeout: 15_000 });

  // Click End.
  await endBtn.click();

  // Poll API until task state is "ended".
  await expect(async () => {
    const resp = await page.request.get("/api/v1/tasks");
    const tasks = await resp.json();
    // Find our task (it may not be the first if tests share state).
    const task = tasks.find(
      (t: { task: string }) => t.task === "e2e end test",
    );
    expect(task).toBeDefined();
    expect(task.state).toBe("ended");
  }).toPass({ timeout: 15_000, intervals: [500] });
});
