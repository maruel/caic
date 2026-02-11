// Claude Code OAuth usage quota fetcher with caching.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/maruel/wmao/backend/internal/server/dto"
)

const (
	usageAPIURL   = "https://api.anthropic.com/api/oauth/usage"
	usageCacheTTL = 30 * time.Second
)

// usageFetcher fetches and caches Claude Code usage quota data.
type usageFetcher struct {
	token  string
	client *http.Client

	mu      sync.Mutex
	cached  *dto.UsageResp
	fetchAt time.Time // when cached was fetched
}

// newUsageFetcher creates a fetcher. Token is resolved from env var
// CLAUDE_OAUTH_TOKEN, falling back to ~/.claude/.credentials.json.
// Returns nil if no token is available.
func newUsageFetcher() *usageFetcher {
	token := os.Getenv("CLAUDE_OAUTH_TOKEN")
	if token == "" {
		token = readCredentialsToken()
	}
	if token == "" {
		slog.Warn("no Claude OAuth token found; usage endpoint disabled")
		return nil
	}
	return &usageFetcher{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// get returns the cached usage data, refreshing if stale.
func (f *usageFetcher) get() *dto.UsageResp {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cached != nil && time.Since(f.fetchAt) < usageCacheTTL {
		return f.cached
	}
	resp, err := f.fetch()
	if err != nil {
		slog.Warn("failed to fetch usage", "err", err)
		return f.cached // return stale on error
	}
	f.cached = resp
	f.fetchAt = time.Now()
	return resp
}

func (f *usageFetcher) fetch() (*dto.UsageResp, error) {
	req, err := http.NewRequest(http.MethodGet, usageAPIURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("usage API returned %d: %s", resp.StatusCode, body)
	}

	var raw struct {
		FiveHour *struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay *struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"seven_day"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode usage: %w", err)
	}

	out := &dto.UsageResp{}
	if raw.FiveHour != nil {
		out.FiveHour = &dto.UsageWindow{
			Utilization: raw.FiveHour.Utilization,
			ResetsAt:    raw.FiveHour.ResetsAt,
		}
	}
	if raw.SevenDay != nil {
		out.SevenDay = &dto.UsageWindow{
			Utilization: raw.SevenDay.Utilization,
			ResetsAt:    raw.SevenDay.ResetsAt,
		}
	}
	return out, nil
}

// readCredentialsToken reads the OAuth token from ~/.claude/.credentials.json.
func readCredentialsToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json")) //nolint:gosec // fixed well-known path
	if err != nil {
		return ""
	}
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal(data, &creds) != nil {
		return ""
	}
	return creds.ClaudeAiOauth.AccessToken
}
