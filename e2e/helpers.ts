// Shared e2e test helpers: typed API client and utilities.
import { test as base, expect, type APIRequestContext } from "@playwright/test";
import type {
  CreateTaskReq,
  CreateTaskResp,
  HarnessInfo,
  InputReq,
  Repo,
  RestartReq,
  StatusResp,
  Task,
} from "../sdk/types.gen";

// ---------------------------------------------------------------------------
// Typed API client wrapping Playwright's APIRequestContext.
// ---------------------------------------------------------------------------

export class APIClient {
  constructor(private request: APIRequestContext) {}

  async listHarnesses(): Promise<HarnessInfo[]> {
    return this.get("/api/v1/harnesses");
  }

  async listRepos(): Promise<Repo[]> {
    return this.get("/api/v1/repos");
  }

  async listTasks(): Promise<Task[]> {
    return this.get("/api/v1/tasks");
  }

  async createTask(req: CreateTaskReq): Promise<CreateTaskResp> {
    return this.post("/api/v1/tasks", req);
  }

  async sendInput(id: string, req: InputReq): Promise<StatusResp> {
    return this.post(`/api/v1/tasks/${id}/input`, req);
  }

  async restartTask(id: string, req: RestartReq): Promise<StatusResp> {
    return this.post(`/api/v1/tasks/${id}/restart`, req);
  }

  async terminateTask(id: string): Promise<StatusResp> {
    return this.post(`/api/v1/tasks/${id}/terminate`);
  }

  async getTask(id: string): Promise<Task | undefined> {
    const tasks = await this.listTasks();
    return tasks.find((t) => t.id === id);
  }

  // -- internal helpers --

  private async get<T>(path: string): Promise<T> {
    const res = await this.request.get(path);
    expect(res.ok(), `GET ${path}: ${res.status()}`).toBe(true);
    return res.json() as Promise<T>;
  }

  private async post<T>(path: string, body?: unknown): Promise<T> {
    const res = await this.request.post(path, { data: body });
    expect(res.ok(), `POST ${path}: ${res.status()}`).toBe(true);
    return res.json() as Promise<T>;
  }
}

// ---------------------------------------------------------------------------
// Fixtures: extends Playwright's base test with an `api` client.
// ---------------------------------------------------------------------------

export const test = base.extend<{ api: APIClient }>({
  api: async ({ request }, use) => {
    await use(new APIClient(request));
  },
});

export { expect };

// ---------------------------------------------------------------------------
// Utility: create a task via API and return its ID.
// ---------------------------------------------------------------------------

export async function createTaskAPI(
  api: APIClient,
  prompt: string,
): Promise<string> {
  const repos = await api.listRepos();
  expect(repos.length).toBeGreaterThan(0);
  const harnesses = await api.listHarnesses();
  expect(harnesses.length).toBeGreaterThan(0);
  const resp = await api.createTask({
    prompt,
    repo: repos[0].path,
    harness: harnesses[0].name,
  });
  expect(resp.id).toBeTruthy();
  return resp.id;
}

// ---------------------------------------------------------------------------
// Utility: poll until a task reaches the expected state.
// ---------------------------------------------------------------------------

export async function waitForTaskState(
  api: APIClient,
  taskId: string,
  state: string,
  timeoutMs = 15_000,
): Promise<Task> {
  let task: Task | undefined;
  await expect(async () => {
    task = await api.getTask(taskId);
    expect(task).toBeTruthy();
    expect(task!.state).toBe(state);
  }).toPass({ timeout: timeoutMs, intervals: [500] });
  return task!;
}
