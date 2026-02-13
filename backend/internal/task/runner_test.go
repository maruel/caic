package task

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maruel/caic/backend/internal/agent"
	"github.com/maruel/ksid"
)

// testWriteFn is a simple WriteFn for testing.
func testWriteFn(w io.Writer, prompt string, logW io.Writer) error {
	msg := struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}{Type: "user"}
	msg.Message.Role = "user"
	msg.Message.Content = prompt
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return err
	}
	if logW != nil {
		_, _ = logW.Write(data)
	}
	return nil
}

// testBackend implements agent.Backend for tests. It launches a "cat" process
// that blocks until stdin is closed. capturedCtx records the context passed
// to Start so tests can assert context lifetime.
type testBackend struct {
	capturedCtx context.Context
}

func (b *testBackend) Name() string { return "test" }

func (b *testBackend) Start(ctx context.Context, _ agent.Options, msgCh chan<- agent.Message, _ io.Writer) (*agent.Session, error) {
	b.capturedCtx = ctx
	cmd := exec.CommandContext(ctx, "cat")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return agent.NewSession(cmd, stdin, stdout, msgCh, nil, testWriteFn), nil
}

func (b *testBackend) AttachRelay(context.Context, string, int64, chan<- agent.Message, io.Writer) (*agent.Session, error) {
	return nil, errors.New("test backend does not support relay")
}

func (b *testBackend) ReadRelayOutput(context.Context, string) ([]agent.Message, int64, error) {
	return nil, 0, errors.New("test backend does not support relay")
}

func (b *testBackend) ParseMessage(line []byte) (agent.Message, error) {
	return agent.ParseMessage(line)
}

func TestRunner(t *testing.T) {
	t.Run("Init", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			clone := initTestRepo(t, "main")
			r := &Runner{
				BaseBranch: "main",
				Dir:        clone,
			}
			if err := r.Init(t.Context()); err != nil {
				t.Fatal(err)
			}
			if r.nextID != 0 {
				t.Errorf("nextID = %d, want 0", r.nextID)
			}
		})
		t.Run("SkipsExisting", func(t *testing.T) {
			clone := initTestRepo(t, "main")
			// Pre-create branches.
			runGit(t, clone, "branch", "caic/w0")
			runGit(t, clone, "branch", "caic/w3")

			r := &Runner{
				BaseBranch: "main",
				Dir:        clone,
			}
			if err := r.Init(t.Context()); err != nil {
				t.Fatal(err)
			}
			if r.nextID != 4 {
				t.Errorf("nextID = %d, want 4", r.nextID)
			}
		})
	})

	t.Run("Kill", func(t *testing.T) {
		t.Run("NoSessionUsesLiveStats", func(t *testing.T) {
			// Simulate an adopted task after server restart: no active session, but
			// live stats were restored from log messages. Kill should fall back to
			// LiveStats for the result cost.
			clone := initTestRepo(t, "main")
			r := &Runner{
				BaseBranch: "main",
				Dir:        clone,
			}

			tk := &Task{
				ID:     ksid.NewID(),
				Prompt: "test",
				Repo:   "org/repo",
				Branch: "main",
				State:  StateRunning,
			}
			tk.InitDoneCh()

			// Restore messages with cost info (simulates RestoreMessages from logs).
			tk.RestoreMessages([]agent.Message{
				&agent.ResultMessage{
					MessageType:  "result",
					TotalCostUSD: 0.42,
					NumTurns:     5,
					DurationMs:   12345,
				},
			})

			// Signal termination immediately.
			tk.Terminate()

			result := r.Kill(t.Context(), tk)
			if result.State != StateTerminated {
				t.Errorf("state = %v, want %v", result.State, StateTerminated)
			}
			if result.CostUSD != 0.42 {
				t.Errorf("CostUSD = %f, want 0.42", result.CostUSD)
			}
			if result.NumTurns != 5 {
				t.Errorf("NumTurns = %d, want 5", result.NumTurns)
			}
			if result.DurationMs != 12345 {
				t.Errorf("DurationMs = %d, want 12345", result.DurationMs)
			}
		})
	})

	t.Run("openLog", func(t *testing.T) {
		t.Run("CreatesFile", func(t *testing.T) {
			dir := t.TempDir()
			logDir := filepath.Join(dir, "logs")
			r := &Runner{LogDir: logDir}
			tk := &Task{ID: ksid.NewID(), Prompt: "test", Repo: "org/repo", Branch: "caic/w0"}
			w, err := r.openLog(tk)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = w.Close() }()
			// Write something and close.
			_, _ = w.Write([]byte("test\n"))
			_ = w.Close()

			entries, err := os.ReadDir(logDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 file, got %d", len(entries))
			}
			name := entries[0].Name()
			want := tk.ID.String() + "-org-repo-caic-w0.jsonl"
			if name != want {
				t.Errorf("filename = %q, want %q", name, want)
			}
		})
	})
}

func TestRestartSession(t *testing.T) {
	logDir := t.TempDir()
	backend := &testBackend{}

	r := &Runner{
		LogDir:       logDir,
		AgentBackend: backend,
	}

	tk := &Task{
		ID:        ksid.NewID(),
		Prompt:    "old prompt",
		Repo:      "org/repo",
		Branch:    "caic/w0",
		Container: "fake-container",
		State:     StateWaiting,
	}

	err := r.RestartSession(t.Context(), tk, "new plan")
	if err != nil {
		t.Fatal(err)
	}
	if tk.State != StateRunning {
		t.Errorf("state = %v, want %v", tk.State, StateRunning)
	}
	if tk.Prompt != "new plan" {
		t.Errorf("prompt = %q, want %q", tk.Prompt, "new plan")
	}

	// The context passed to AgentBackend.Start must still be valid after
	// RestartSession returns (it must not be a request-scoped context).
	select {
	case <-backend.capturedCtx.Done():
		t.Error("context passed to AgentBackend was canceled; must use a long-lived context")
	default:
	}

	// Verify the session is functional: wait briefly and check the context
	// is still alive (not canceled by a short-lived HTTP request context).
	time.Sleep(50 * time.Millisecond)
	select {
	case <-backend.capturedCtx.Done():
		t.Error("context was canceled shortly after RestartSession returned")
	default:
	}

	// Clean up: close the session.
	tk.CloseSession()
}

// initTestRepo creates a bare "remote" and a local clone with one commit on
// baseBranch. Returns the clone directory. origin points to the bare repo so
// git fetch/push work locally.
func initTestRepo(t *testing.T, baseBranch string) string { //nolint:unparam // baseBranch is parameterized for clarity.
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "remote.git")
	clone := filepath.Join(dir, "clone")

	runGit(t, "", "init", "--bare", bare)
	runGit(t, "", "init", clone)
	runGit(t, clone, "config", "user.name", "Test")
	runGit(t, clone, "config", "user.email", "test@test.com")
	runGit(t, clone, "checkout", "-b", baseBranch)

	if err := os.WriteFile(filepath.Join(clone, "README.md"), []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", ".")
	runGit(t, clone, "commit", "-m", "init")
	runGit(t, clone, "remote", "add", "origin", bare)
	runGit(t, clone, "push", "-u", "origin", baseBranch)
	return clone
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
