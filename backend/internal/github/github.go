// Package github provides a minimal GitHub REST API client for PR creation
// and CI check-run polling. Uses net/http directly; no extra dependencies.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

// Client is a minimal GitHub API client authenticated with a personal access token.
type Client struct {
	Token string
}

// PR holds the fields of a GitHub pull request returned after creation.
type PR struct {
	Number  int
	HeadSHA string
}

// CheckRunStatus is the status of a GitHub check run.
type CheckRunStatus string

// Check run status values.
const (
	CheckRunStatusQueued     CheckRunStatus = "queued"
	CheckRunStatusInProgress CheckRunStatus = "in_progress"
	CheckRunStatusCompleted  CheckRunStatus = "completed"
)

// CheckRunConclusion is the conclusion of a completed GitHub check run.
// Empty when not yet completed.
type CheckRunConclusion string

// Check run conclusion values.
const (
	CheckRunConclusionSuccess        CheckRunConclusion = "success"
	CheckRunConclusionFailure        CheckRunConclusion = "failure"
	CheckRunConclusionNeutral        CheckRunConclusion = "neutral"
	CheckRunConclusionSkipped        CheckRunConclusion = "skipped"
	CheckRunConclusionCancelled      CheckRunConclusion = "cancelled"
	CheckRunConclusionTimedOut       CheckRunConclusion = "timed_out"
	CheckRunConclusionActionRequired CheckRunConclusion = "action_required"
	CheckRunConclusionStale          CheckRunConclusion = "stale"
)

// CheckRun describes a single GitHub Actions check run.
type CheckRun struct {
	ID         int64
	Name       string
	Status     CheckRunStatus
	Conclusion CheckRunConclusion
}

// createPRRequest is the JSON body for POST /repos/{owner}/{repo}/pulls.
type createPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// createPRResponse is the relevant subset of the GitHub PR creation response.
type createPRResponse struct {
	Number int `json:"number"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// checkRunsResponse is the relevant subset of the GitHub check-runs list response.
type checkRunsResponse struct {
	CheckRuns []struct {
		ID         int64              `json:"id"`
		Name       string             `json:"name"`
		Status     CheckRunStatus     `json:"status"`
		Conclusion CheckRunConclusion `json:"conclusion"`
	} `json:"check_runs"`
}

// CreatePR creates a pull request on GitHub and returns its metadata.
func (c *Client) CreatePR(ctx context.Context, owner, repo, head, base, title, body string) (PR, error) {
	payload, err := json.Marshal(createPRRequest{Title: title, Body: body, Head: head, Base: base})
	if err != nil {
		return PR{}, err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return PR{}, err
	}
	c.setHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return PR{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return PR{}, err
	}
	if resp.StatusCode != http.StatusCreated {
		return PR{}, fmt.Errorf("github create PR: status %d: %s", resp.StatusCode, data)
	}
	var r createPRResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return PR{}, err
	}
	return PR{Number: r.Number, HeadSHA: r.Head.SHA}, nil
}

// GetCheckRuns returns all check runs for the given commit SHA.
func (c *Client) GetCheckRuns(ctx context.Context, owner, repo, sha string) ([]CheckRun, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/check-runs", owner, repo, sha)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github get check-runs: status %d: %s", resp.StatusCode, data)
	}
	var r checkRunsResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	runs := make([]CheckRun, len(r.CheckRuns))
	for i, cr := range r.CheckRuns {
		runs[i] = CheckRun{
			ID:         cr.ID,
			Name:       cr.Name,
			Status:     cr.Status,
			Conclusion: cr.Conclusion,
		}
	}
	return runs, nil
}

// httpsRe matches "https://github.com/owner/repo" (with optional .git suffix).
var httpsRe = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)

// sshRe matches "git@github.com:owner/repo" (with optional .git suffix).
var sshRe = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

// ParseRemoteURL extracts the owner and repo name from a GitHub remote URL.
// Handles both HTTPS ("https://github.com/owner/repo") and SSH
// ("git@github.com:owner/repo") formats, with or without a ".git" suffix.
func ParseRemoteURL(rawURL string) (owner, repo string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if m := httpsRe.FindStringSubmatch(rawURL); m != nil {
		return m[1], m[2], nil
	}
	if m := sshRe.FindStringSubmatch(rawURL); m != nil {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("not a github remote URL: %q", rawURL)
}

// RemoteURL returns the URL of the "origin" remote for the git repository at dir.
func RemoteURL(ctx context.Context, dir string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin").Output() //nolint:gosec // dir is a trusted repo path
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
}
