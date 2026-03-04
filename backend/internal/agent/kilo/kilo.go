// Package kilo implements agent.Backend for Kilo Code.
//
// Kilo Code exposes an HTTP+SSE API via `kilo serve`. A Python bridge script
// (bridge.py) runs as the relay subprocess, handling process management, SSE
// I/O, permission auto-approval, and deduplication. It emits native kilo SSE
// event JSON on stdout, which ParseMessage translates into agent.Message types.
package kilo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/caic-xyz/caic/backend/internal/agent"
)

const bridgeScriptPath = agent.RelayDir + "/kilo_bridge.py"

// Backend implements agent.Backend for Kilo Code.
type Backend struct {
	agent.Base
	modelsMu sync.RWMutex
}

var _ agent.Backend = (*Backend)(nil)

var defaultModels = []string{
	"anthropic/claude-opus-4.6",
	"anthropic/claude-sonnet-4.6",
	"google/gemini-3.1-pro-preview",
	"google/gemini-3-flash-preview",
	"openai/gpt-5.3-codex",
}

// New creates a Kilo Code backend with wire format and parser configured.
// Models default to a small hardcoded list; call SetModels to update
// asynchronously after discovery.
func New() *Backend {
	b := &Backend{}
	b.Base = agent.Base{
		HarnessID:     agent.Kilo,
		ModelList:     defaultModels,
		ContextWindow: 200_000,
		Parse:         ParseMessage,
	}
	b.Wire = b
	return b
}

// Models returns the available model list (thread-safe).
func (b *Backend) Models() []string {
	b.modelsMu.RLock()
	defer b.modelsMu.RUnlock()
	return b.ModelList
}

// SetModels replaces the model list (thread-safe).
func (b *Backend) SetModels(models []string) {
	b.modelsMu.Lock()
	defer b.modelsMu.Unlock()
	b.ModelList = models
}

// Start deploys relay and bridge scripts, then launches via relay serve-attach.
func (b *Backend) Start(ctx context.Context, opts *agent.Options, msgCh chan<- agent.Message, logW io.Writer) (*agent.Session, error) {
	if err := deployBridge(ctx, opts.Container); err != nil {
		return nil, err
	}
	return agent.StartRelay(ctx, opts, buildBridgeArgs(opts), msgCh, logW, b)
}

// WritePrompt writes a single user message to the bridge's stdin.
// The bridge reads plain text lines (like Gemini CLI).
func (*Backend) WritePrompt(w io.Writer, p agent.Prompt, logW io.Writer) error {
	return agent.PlainTextWritePrompt(w, p, logW)
}

// deployBridge uploads the bridge script into the container. Idempotent.
func deployBridge(ctx context.Context, container string) error {
	cmd := exec.CommandContext(ctx, "ssh", container, //nolint:gosec // container is not user-controlled
		"mkdir -p "+agent.RelayDir+" && cat > "+bridgeScriptPath)
	cmd.Stdin = bytes.NewReader(BridgeScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("deploy kilo bridge: %w: %s", err, out)
	}
	return nil
}

// buildBridgeArgs constructs the command to run the bridge script.
func buildBridgeArgs(opts *agent.Options) []string {
	args := []string{"python3", "-u", bridgeScriptPath}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	return args
}
