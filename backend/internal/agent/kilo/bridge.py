# Bridge between relay stdin/stdout NDJSON and kilo serve HTTP+SSE.
#
# The relay daemon manages this script as its subprocess. It handles process
# management (start kilo serve, session lifecycle), SSE I/O, permission
# auto-approval, deduplication, and shutdown. It emits native kilo SSE event
# JSON on stdout; the Go-side ParseMessage handles format translation.
#
#   Go Backend <-SSH-> relay.py <-stdin/stdout-> bridge.py <-HTTP+SSE-> kilo serve
#
# Key API details:
#   POST /session      — create session (body: {}, NOT empty)
#   POST /session/:id/prompt_async — fire-and-forget prompt (returns 204)
#   GET  /global/event — SSE stream, events wrapped as {"payload": {"type":..., "properties":...}}
#   POST /permission/:id/reply — auto-approve with {"reply": "always"}
#   POST /global/dispose — graceful shutdown
#   All models route through OpenRouter: providerID="openrouter", modelID="provider/model"
#
# Usage: python3 bridge.py [--model provider/model]
#
# Uses only Python stdlib (no pip dependencies).

import argparse
import atexit
import ctypes
import ctypes.util
import json
import os
import signal
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def emit(obj):
    """Write a NDJSON line to stdout and flush."""
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def log(msg):
    """Write a diagnostic message to stderr."""
    print(f"kilo_bridge: {msg}", file=sys.stderr, flush=True)


def http_post(url, body=None, password=None):
    """POST JSON to url, return parsed JSON response (or None for 204)."""
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if password:
        import base64

        cred = base64.b64encode(f":{password}".encode()).decode()
        req.add_header("Authorization", f"Basic {cred}")
    with urllib.request.urlopen(req) as resp:
        if resp.status == 204:
            return None
        return json.loads(resp.read())


def http_get_stream(url, password=None):
    """Open a GET request and return the response object for streaming."""
    req = urllib.request.Request(url)
    if password:
        import base64

        cred = base64.b64encode(f":{password}".encode()).decode()
        req.add_header("Authorization", f"Basic {cred}")
    return urllib.request.urlopen(req)


def _set_pdeathsig():
    """Called in child process via preexec_fn to auto-SIGTERM on parent death."""
    try:
        libc_name = ctypes.util.find_library("c")
        if libc_name:
            libc = ctypes.CDLL(libc_name, use_errno=True)
            PR_SET_PDEATHSIG = 1
            libc.prctl(PR_SET_PDEATHSIG, signal.SIGTERM)
    except Exception:
        pass


# ---------------------------------------------------------------------------
# SSE reader
# ---------------------------------------------------------------------------


def read_sse_events(resp):
    """Yield parsed SSE data events from an HTTP response stream."""
    buf = b""
    while True:
        chunk = resp.read(1)
        if not chunk:
            break
        buf += chunk
        while b"\n\n" in buf:
            raw_event, buf = buf.split(b"\n\n", 1)
            data_parts = []
            for line in raw_event.decode("utf-8", errors="replace").split("\n"):
                if line.startswith("data:"):
                    data_parts.append(line[len("data:") :].strip())
            if data_parts:
                text = "\n".join(data_parts)
                try:
                    yield json.loads(text)
                except json.JSONDecodeError:
                    log(f"SSE decode error: {text[:200]}")


# ---------------------------------------------------------------------------
# Bridge
# ---------------------------------------------------------------------------


class KiloBridge:
    def __init__(self, model):
        self.model = model
        self.base_url = None
        self.session_id = None
        self.password = os.environ.get("KILO_SERVER_PASSWORD")
        self.kilo_proc = None
        # Track emitted (kind, part_id) to deduplicate SSE events.
        self.emitted = set()

    def start_kilo(self):
        """Start kilo serve and extract the listen URL."""
        args = ["kilo", "serve", "--port", "0", "--hostname", "127.0.0.1"]
        preexec = _set_pdeathsig if sys.platform == "linux" else None
        self.kilo_proc = subprocess.Popen(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            preexec_fn=preexec,
        )
        atexit.register(self._kill_kilo)

        # Read stdout lines until we see the listen URL.
        for line in self.kilo_proc.stdout:
            line = line.strip()
            log(f"kilo: {line}")
            if "listening on" in line.lower():
                for word in line.split():
                    if word.startswith("http://") or word.startswith("https://"):
                        self.base_url = word.rstrip("/")
                        break
                if self.base_url:
                    break

        if not self.base_url:
            log("ERROR: failed to extract kilo listen URL")
            sys.exit(1)
        log(f"kilo serve ready at {self.base_url}")

        # Drain remaining kilo stdout in background.
        threading.Thread(target=self._drain_kilo, daemon=True).start()

    def _drain_kilo(self):
        for line in self.kilo_proc.stdout:
            log(f"kilo: {line.rstrip()}")

    def _kill_kilo(self):
        if self.kilo_proc and self.kilo_proc.poll() is None:
            log("killing kilo serve")
            try:
                self.kilo_proc.terminate()
                self.kilo_proc.wait(timeout=5)
            except Exception:
                try:
                    self.kilo_proc.kill()
                except Exception:
                    pass

    def create_session(self):
        """Create a kilo session via POST /session."""
        resp = http_post(f"{self.base_url}/session", {}, self.password)
        self.session_id = resp["id"]
        log(f"session created: {self.session_id}")

    def send_prompt(self, text):
        """Send a prompt to kilo via POST /session/:id/prompt_async."""
        body = {"parts": [{"type": "text", "text": text}]}
        if self.model:
            body["model"] = {"providerID": "openrouter", "modelID": self.model}
        try:
            http_post(
                f"{self.base_url}/session/{self.session_id}/prompt_async",
                body,
                self.password,
            )
        except urllib.error.HTTPError as e:
            log(f"prompt_async error: {e.code} {e.read().decode()[:500]}")

    def auto_approve(self, permission_id):
        """Auto-approve a permission request (equivalent to --yolo)."""
        try:
            http_post(
                f"{self.base_url}/permission/{permission_id}/reply",
                {"reply": "always"},
                self.password,
            )
        except urllib.error.HTTPError as e:
            log(f"permission reply error: {e.code}")

    # -----------------------------------------------------------------------
    # SSE event handling
    # -----------------------------------------------------------------------

    def handle_sse_event(self, event):
        # SSE events are wrapped: {"payload": {"type": ..., "properties": ...}}
        payload = event.get("payload", event)
        event = payload
        etype = event.get("type", "")

        if etype == "permission.asked":
            props = event.get("properties", event)
            pid = props.get("id")
            if pid:
                self.auto_approve(pid)
            return

        if etype == "message.part.updated":
            self._handle_part_updated(event)
            return

        if etype == "message.part.delta":
            self._handle_part_delta(event)
            return

        if etype == "session.error":
            emit(event)
            return

        if etype == "session.turn.close":
            emit(event)
            return

    def _handle_part_updated(self, event):
        props = event.get("properties", {})
        part = props.get("part") or props
        part_type = part.get("type", "")
        part_id = part.get("id", "")

        if part_type == "text":
            t = part.get("time", {})
            if t.get("end") and ("text", part_id) not in self.emitted:
                self.emitted.add(("text", part_id))
                emit(event)

        elif part_type == "tool":
            state = part.get("state", {})
            status = state.get("status", "")

            if status == "running" and ("tool_use", part_id) not in self.emitted:
                self.emitted.add(("tool_use", part_id))
                emit(event)

            elif (
                status in ("completed", "error")
                and (
                    "tool_result",
                    part_id,
                )
                not in self.emitted
            ):
                self.emitted.add(("tool_result", part_id))
                emit(event)

        elif part_type == "step-finish":
            if ("step_finish", part_id) not in self.emitted:
                self.emitted.add(("step_finish", part_id))
                emit(event)

        elif part_type == "reasoning":
            t = part.get("time", {})
            if t.get("end") and ("reasoning", part_id) not in self.emitted:
                self.emitted.add(("reasoning", part_id))
                emit(event)

        elif part_type == "step-start":
            if ("step_start", part_id) not in self.emitted:
                self.emitted.add(("step_start", part_id))
                emit(event)

    def _handle_part_delta(self, event):
        props = event.get("properties", {})
        delta = props.get("delta", "")
        if delta:
            emit(event)

    def run_sse_loop(self):
        """Connect to SSE and process events. Reconnects on error."""
        url = f"{self.base_url}/global/event"
        while True:
            try:
                resp = http_get_stream(url, self.password)
                for event in read_sse_events(resp):
                    self.handle_sse_event(event)
            except Exception as e:
                log(f"SSE error: {e}")
                if self.kilo_proc and self.kilo_proc.poll() is not None:
                    log("kilo serve exited, stopping SSE loop")
                    return
                time.sleep(1)

    def dispose(self):
        """Gracefully shut down kilo serve."""
        if self.base_url:
            try:
                http_post(f"{self.base_url}/global/dispose", {}, self.password)
            except Exception:
                pass

    def run(self):
        self.start_kilo()
        self.create_session()

        # Emit init before reading any prompts.
        emit(
            {
                "type": "system",
                "subtype": "init",
                "session_id": self.session_id,
                "model": self.model or "",
            }
        )

        # Start SSE reader in background.
        threading.Thread(target=self.run_sse_loop, daemon=True).start()

        # Read plain text prompts from stdin (one per line).
        try:
            for line in sys.stdin:
                line = line.rstrip("\n")
                if not line:
                    continue
                if "\x00" in line:
                    log("received null byte, shutting down")
                    break
                self.send_prompt(line)
        except EOFError:
            pass
        finally:
            self.dispose()


def main():
    parser = argparse.ArgumentParser(description="Kilo bridge for caic relay")
    parser.add_argument("--model", default="", help="Model in provider/id format")
    args = parser.parse_args()
    KiloBridge(args.model).run()


if __name__ == "__main__":
    main()
