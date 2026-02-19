// Request validation methods (excluded from tygo generation).
package dto

// Validatable is implemented by request types that can validate their fields.
type Validatable interface {
	Validate() error
}

// Validate is a no-op for empty requests.
func (EmptyReq) Validate() error { return nil }

// Validate checks that prompt or images are provided.
func (r *InputReq) Validate() error {
	if r.Prompt == "" && len(r.Images) == 0 {
		return BadRequest("prompt or images required")
	}
	return validateImages(r.Images)
}

// Validate is a no-op; prompt is optional (read from container plan file if empty).
func (r *RestartReq) Validate() error { return nil }

// Validate is a no-op for sync requests.
func (SyncReq) Validate() error { return nil }

// Validate checks that prompt, repo, and harness are valid.
func (r *CreateTaskReq) Validate() error {
	if r.Prompt == "" && len(r.Images) == 0 {
		return BadRequest("prompt or images required")
	}
	if r.Repo == "" {
		return BadRequest("repo is required")
	}
	if r.Harness == "" {
		return BadRequest("harness is required")
	}
	return validateImages(r.Images)
}

// allowedImageTypes is the set of MIME types accepted for image uploads.
var allowedImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// validateImages checks that each ImageData entry has a valid media type and non-empty data.
func validateImages(images []ImageData) error {
	for _, img := range images {
		if img.MediaType == "" {
			return BadRequest("image mediaType is required")
		}
		if !allowedImageTypes[img.MediaType] {
			return BadRequest("unsupported image mediaType: " + img.MediaType)
		}
		if img.Data == "" {
			return BadRequest("image data is required")
		}
	}
	return nil
}
