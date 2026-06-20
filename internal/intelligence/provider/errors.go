package provider

import "errors"

// Typed errors returned by providers. The router uses these to decide whether
// a failure is eligible for fallback: validation, unsupported, and
// invalid-request errors are not retried against another provider, while
// unavailable-provider and transient errors are.
var (
	// ErrUnavailableProvider is returned when a provider cannot be reached or
	// is not registered. Eligible for fallback.
	ErrUnavailableProvider = errors.New("provider unavailable")

	// ErrInvalidResponse is returned when a provider responds but the payload
	// cannot be parsed or does not satisfy the expected shape. Not eligible for
	// fallback by default, since a second provider will not fix a malformed
	// local response; callers may still choose to retry.
	ErrInvalidResponse = errors.New("invalid provider response")

	// ErrUnsupportedOperation is returned when a provider does not implement
	// the requested operation (for example, an embedding call against a
	// completion-only provider). Not eligible for fallback.
	ErrUnsupportedOperation = errors.New("unsupported provider operation")
)