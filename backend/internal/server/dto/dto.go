// Package dto provides shared API infrastructure (errors, validation interface)
// used across all API versions. Version-specific types live in sub-packages
// (e.g. dto/v1).
package dto

// Validatable is implemented by request types that can validate their fields.
type Validatable interface {
	Validate() error
}

// EmptyReq is used for endpoints that take no request body.
type EmptyReq struct{}

// Validate is a no-op for empty requests.
func (EmptyReq) Validate() error { return nil }
