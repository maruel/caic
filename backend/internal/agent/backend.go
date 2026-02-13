package agent

import (
	"context"
	"io"
)

// Backend launches and communicates with a coding agent process.
// Each implementation translates its native wire format into the shared
// Message types so the rest of the system (task, eventconv, SSE, frontend)
// remains agent-agnostic.
type Backend interface {
	// Start launches the agent in the given container. Messages are emitted
	// to msgCh as normalized agent.Message values. logW receives raw
	// wire-format lines for debugging/replay.
	Start(ctx context.Context, opts Options, msgCh chan<- Message, logW io.Writer) (*Session, error)

	// AttachRelay connects to an already-running relay daemon in the
	// container. The offset parameter specifies the byte offset into
	// output.jsonl to replay from (use 0 for full replay).
	AttachRelay(ctx context.Context, container string, offset int64, msgCh chan<- Message, logW io.Writer) (*Session, error)

	// ReadRelayOutput reads the complete output.jsonl from the container's
	// relay and parses it into Messages. Also returns the byte count for
	// use as an offset in AttachRelay.
	ReadRelayOutput(ctx context.Context, container string) ([]Message, int64, error)

	// ParseMessage decodes a single wire-format line into a normalized
	// Message. Used for log replay (load.go).
	ParseMessage(line []byte) (Message, error)

	// Harness returns the harness identifier ("claude", "gemini", etc.)
	Harness() Harness
}
