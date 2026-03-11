// Tests for repo dropdown ordering after clone and task creation.
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@solidjs/testing-library";
import userEvent from "@testing-library/user-event";
import type { Repo, PreferencesResp, HarnessInfo } from "@sdk/types.gen";

const navigateMock = vi.fn();

vi.mock("@solidjs/router", () => ({
  useNavigate: () => navigateMock,
  useLocation: () => ({ pathname: "/" }),
}));

vi.mock("./api", () => ({
  listRepos: vi.fn(),
  getPreferences: vi.fn(),
  listHarnesses: vi.fn(),
  getConfig: vi.fn(),
  getUsage: vi.fn(),
  listRepoBranches: vi.fn(),
  cloneRepo: vi.fn(),
  createTask: vi.fn(),
  terminateTask: vi.fn(),
}));

vi.mock("./AuthContext", () => ({
  // eslint-disable-next-line solid/reactivity
  AuthProvider: (props: { children: unknown }) => props.children,
  useAuth: () => ({
    ready: () => true,
    providers: () => [],
    user: () => null,
    logout: async () => {},
  }),
}));

// Stub EventSource to prevent real SSE connections.
// FakeEventSource captures message listeners so tests can push SSE events.
type MessageListener = (e: { data: string }) => void;
const fakeESListeners: MessageListener[] = [];

class FakeEventSource {
  addEventListener = vi.fn((type: string, handler: MessageListener) => {
    if (type === "message") fakeESListeners.push(handler);
  });
  close = vi.fn();
  onerror: ((e: Event) => void) | null = null;
}
vi.stubGlobal("EventSource", FakeEventSource);

function dispatchSSE(data: unknown) {
  const payload = { data: JSON.stringify(data) };
  fakeESListeners.forEach((fn) => fn(payload));
}

// Imports must follow vi.mock declarations.
import App from "./App";
import * as api from "./api";

const repoA: Repo = { path: "repos/a", baseBranch: "main", remoteURL: "" };
const repoB: Repo = { path: "repos/b", baseBranch: "main", remoteURL: "" };
const newRepo: Repo = { path: "repos/new", baseBranch: "main", remoteURL: "" };

beforeEach(() => {
  vi.clearAllMocks();
  navigateMock.mockClear();
  fakeESListeners.length = 0;
  vi.mocked(api.listRepos).mockResolvedValue([repoA, repoB]);
  vi.mocked(api.getPreferences).mockResolvedValue({
    repositories: [{ path: "repos/a" }],
    models: {},
    harness: "",
    baseImage: "",
  } as PreferencesResp);
  vi.mocked(api.listHarnesses).mockResolvedValue([
    { name: "claude", models: [], supportsImages: false },
  ] as HarnessInfo[]);
  vi.mocked(api.getConfig).mockRejectedValue(new Error("no config"));
  vi.mocked(api.getUsage).mockRejectedValue(new Error("no usage"));
  vi.mocked(api.listRepoBranches).mockResolvedValue({ branches: ["main", "dev"] });
  vi.mocked(api.cloneRepo).mockResolvedValue(newRepo);
  vi.mocked(api.createTask).mockResolvedValue({ id: "task1" });
});

describe("App repo select: No repository", () => {
  it("stays on No repository after manual selection", async () => {
    const user = userEvent.setup();
    render(() => <App />);

    // Wait for initial load: repos/a is the recent repo and should be selected.
    await waitFor(() => {
      expect((screen.getByTestId("repo-select") as HTMLSelectElement).value).toBe("repos/a");
    });

    // User explicitly selects "No repository".
    const sel = screen.getByTestId("repo-select") as HTMLSelectElement;
    await user.selectOptions(sel, "");

    // The selection must remain "No repository" (value="").
    expect(sel.value).toBe("");
  });

  it("stays on No repository after repos SSE event updates CI status", async () => {
    const user = userEvent.setup();
    render(() => <App />);

    await waitFor(() => {
      expect((screen.getByTestId("repo-select") as HTMLSelectElement).value).toBe("repos/a");
    });

    const sel = screen.getByTestId("repo-select") as HTMLSelectElement;
    await user.selectOptions(sel, "");
    expect(sel.value).toBe("");

    // Simulate a "repos" SSE event (e.g. CI status update) which triggers setRepos.
    const repoAUpdated: Repo = { path: "repos/a", baseBranch: "main", remoteURL: "", defaultBranchCIStatus: "success" as const };
    dispatchSSE({ kind: "repos", repos: [repoAUpdated] });

    await waitFor(() => {
      // Selection must remain "No repository" — not revert to the first repo.
      expect(sel.value).toBe("");
    });
  });

  it("creates task without repos when No repository is selected", async () => {
    const user = userEvent.setup();
    render(() => <App />);

    await waitFor(() => {
      expect((screen.getByTestId("repo-select") as HTMLSelectElement).value).toBe("repos/a");
    });

    const sel = screen.getByTestId("repo-select") as HTMLSelectElement;
    await user.selectOptions(sel, "");

    await user.type(screen.getByTestId("prompt-input"), "do something");
    await user.click(screen.getByTestId("submit-task"));

    await waitFor(() => expect(api.createTask).toHaveBeenCalledOnce());
    const call = vi.mocked(api.createTask).mock.calls[0][0];
    expect(call.repos).toBeUndefined();
  });
});

describe("App repo select ordering", () => {
  it("defaults to the last-used repo from preferences on load", async () => {
    // getPreferences returns repos/b as MRU first (last used to create a task).
    // Recent group is sorted alphabetically so repos/a appears first visually,
    // but the selected value must still be repos/b.
    vi.mocked(api.getPreferences).mockResolvedValue({
      repositories: [{ path: "repos/b" }, { path: "repos/a" }],
      models: {},
      harness: "",
      baseImage: "",
    } as PreferencesResp);
    render(() => <App />);

    await waitFor(() => {
      const sel = screen.getByTestId("repo-select") as HTMLSelectElement;
      expect(sel.value).toBe("repos/b");
    });
  });

  it("cloned repo appears in All repositories (not Recent) before first task", async () => {
    const user = userEvent.setup();
    render(() => <App />);

    // Wait for initial load: A is recent so optgroups are shown.
    await waitFor(() => {
      const sel = screen.getByTestId("repo-select");
      expect(sel.querySelector("optgroup[label='Recent']")).toBeInTheDocument();
    });

    // Clone a new repo.
    await user.click(screen.getByTestId("clone-toggle"));
    await user.type(screen.getByTestId("clone-url"), "https://github.com/org/new.git");
    await user.click(screen.getByTestId("clone-submit"));

    // Wait for clone to complete and dialog to close.
    await waitFor(() => expect(screen.queryByTestId("clone-url")).not.toBeInTheDocument());

    const repoSelect = screen.getByTestId("repo-select");

    // The cloned repo must NOT appear in the Recent optgroup.
    const recentGroup = repoSelect.querySelector("optgroup[label='Recent']");
    expect(recentGroup).toBeInTheDocument();
    const recentPaths = Array.from(recentGroup?.querySelectorAll("option") ?? []).map(
      (o) => (o as HTMLOptionElement).value,
    );
    expect(recentPaths).not.toContain(newRepo.path);

    // The cloned repo MUST appear in the All repositories optgroup.
    const allGroup = repoSelect.querySelector("optgroup[label='All repositories']");
    expect(allGroup).toBeInTheDocument();
    const allPaths = Array.from(allGroup?.querySelectorAll("option") ?? []).map(
      (o) => (o as HTMLOptionElement).value,
    );
    expect(allPaths).toContain(newRepo.path);
  });

  it("cloned repo moves to Recent after first task", async () => {
    const user = userEvent.setup();
    render(() => <App />);

    await waitFor(() => {
      const sel = screen.getByTestId("repo-select");
      expect(sel.querySelector("optgroup[label='Recent']")).toBeInTheDocument();
    });

    // Clone a new repo.
    await user.click(screen.getByTestId("clone-toggle"));
    await user.type(screen.getByTestId("clone-url"), "https://github.com/org/new.git");
    await user.click(screen.getByTestId("clone-submit"));
    await waitFor(() => expect(screen.queryByTestId("clone-url")).not.toBeInTheDocument());

    // Submit a task for the new repo (it should already be selected after clone).
    await user.type(screen.getByTestId("prompt-input"), "do something");
    await user.click(screen.getByTestId("submit-task"));
    await waitFor(() => expect(api.createTask).toHaveBeenCalledOnce());

    const repoSelect = screen.getByTestId("repo-select");
    const recentGroup = repoSelect.querySelector("optgroup[label='Recent']");
    const recentPaths = Array.from(recentGroup?.querySelectorAll("option") ?? []).map(
      (o) => (o as HTMLOptionElement).value,
    );
    expect(recentPaths).toContain(newRepo.path);
  });
});
