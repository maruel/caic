// Exported request and response types for the wmao API.
package dto

// RepoJSON is the JSON representation of a discovered repo.
type RepoJSON struct {
	Path       string `json:"path"`
	BaseBranch string `json:"baseBranch"`
}

// TaskJSON is the JSON representation sent to the frontend.
type TaskJSON struct {
	ID         int     `json:"id"`
	Task       string  `json:"task"`
	Repo       string  `json:"repo"`
	Branch     string  `json:"branch"`
	Container  string  `json:"container"`
	State      string  `json:"state"`
	DiffStat   string  `json:"diffStat"`
	CostUSD    float64 `json:"costUSD"`
	DurationMs int64   `json:"durationMs"`
	NumTurns   int     `json:"numTurns"`
	Error      string  `json:"error,omitempty"`
	Result     string  `json:"result,omitempty"`
}

// StatusResp is a common response for mutation endpoints.
type StatusResp struct {
	Status string `json:"status"`
}

// CreateTaskReq is the request body for POST /api/v1/tasks.
type CreateTaskReq struct {
	Prompt string `json:"prompt"`
	Repo   string `json:"repo"`
}

// InputReq is the request body for POST /api/v1/tasks/{id}/input.
type InputReq struct {
	Prompt string `json:"prompt"`
}

// EmptyReq is used for endpoints that take no request body.
type EmptyReq struct{}
