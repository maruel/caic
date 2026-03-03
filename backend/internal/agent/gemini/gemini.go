// Package gemini implements agent.Backend for Gemini CLI.
package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"

	"github.com/caic-xyz/caic/backend/internal/agent"
)

// Backend implements agent.Backend for Gemini CLI.
type Backend struct {
	agent.Base
}

var _ agent.Backend = (*Backend)(nil)

// New creates a Gemini CLI backend with wire format and parser configured.
func New() *Backend {
	b := &Backend{}
	b.Base = agent.Base{
		HarnessID:     agent.Gemini,
		ModelList:     []string{"gemini-3.1-pro", "gemini-3-flash"},
		ContextWindow: 1_000_000,
		Parse:         ParseMessage,
	}
	b.Wire = b
	return b
}

// Wire is the wire format for Gemini CLI (stream-json over stdin/stdout).
var Wire agent.WireFormat = New()

// Start launches a Gemini CLI process via the relay daemon in the given
// container.
func (b *Backend) Start(ctx context.Context, opts *agent.Options, msgCh chan<- agent.Message, logW io.Writer) (*agent.Session, error) {
	if opts.Dir == "" {
		return nil, errors.New("opts.Dir is required")
	}
	if err := agent.DeployRelay(ctx, opts.Container); err != nil {
		return nil, err
	}

	geminiArgs := buildArgs(opts)

	sshArgs := make([]string, 0, 7+len(geminiArgs))
	sshArgs = append(sshArgs, opts.Container, "python3", agent.RelayScriptPath, "serve-attach", "--dir", opts.Dir, "--")
	sshArgs = append(sshArgs, geminiArgs...)

	slog.Info("gemini: launching via relay", "container", opts.Container, "args", geminiArgs)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...) //nolint:gosec // args are not user-controlled.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = &agent.SlogWriter{Prefix: "relay serve-attach", Container: opts.Container}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start relay: %w", err)
	}

	log := slog.With("container", opts.Container)
	s := agent.NewSession(cmd, stdin, stdout, msgCh, logW, b, log)
	if opts.InitialPrompt.Text != "" {
		if err := s.Send(opts.InitialPrompt); err != nil {
			s.Close()
			return nil, fmt.Errorf("write prompt: %w", err)
		}
	}
	return s, nil
}

// WritePrompt writes a single user message to Gemini CLI's stdin.
// Gemini CLI in -p mode reads plain text lines from stdin. Images are ignored.
func (*Backend) WritePrompt(w io.Writer, p agent.Prompt, logW io.Writer) error {
	return agent.PlainTextWritePrompt(w, p, logW)
}

// buildArgs constructs the Gemini CLI arguments.
func buildArgs(opts *agent.Options) []string {
	args := []string{
		"gemini", "-p",
		"--output-format", "stream-json",
		"--yolo",
	}
	if opts.Model != "" {
		args = append(args, "-m", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--resume", opts.ResumeSessionID)
	}
	return args
}
