// Request validation methods (excluded from tygo generation).
package dto

// Validatable is implemented by request types that can validate their fields.
type Validatable interface {
	Validate() error
}

// Validate is a no-op for empty requests.
func (EmptyReq) Validate() error { return nil }

// Validate checks that the prompt is non-empty.
func (r *InputReq) Validate() error {
	if r.Prompt == "" {
		return BadRequest("prompt is required")
	}
	return nil
}

// Validate is a no-op; prompt is optional (read from container plan file if empty).
func (r *RestartReq) Validate() error { return nil }

// Validate is a no-op for sync requests.
func (SyncReq) Validate() error { return nil }

// Validate checks that prompt, repo, and harness are valid.
func (r *CreateTaskReq) Validate() error {
	if r.Prompt == "" {
		return BadRequest("prompt is required")
	}
	if r.Repo == "" {
		return BadRequest("repo is required")
	}
	if r.Harness == "" {
		return BadRequest("harness is required")
	}
	return nil
}
