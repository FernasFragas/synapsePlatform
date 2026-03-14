package auth

import (
	"errors"
	"fmt"
	"synapsePlatform/internal"

	"github.com/golang-jwt/jwt/v5"
)

const (
	ErrTypeTokenExpired internal.TypeOfError = iota
	ErrTypeTokenNotYetValid
	ErrTypeTokenInvalid
	ErrTypeTokenWrongIssuer
	ErrTypeTokenWrongAudience
	ErrTypeMissingIdentity
	ErrTypeConfiguration
	ErrTypeNoIdentity
)

// ErrorOccurredBecauseOf is the specific reason within a category.
type ErrorOccurredBecauseOf string

const (
	ErrTokenExpired            ErrorOccurredBecauseOf = "token has expired"
	ErrTokenNotYetValid        ErrorOccurredBecauseOf = "token is not yet valid"
	ErrTokenSignatureInvalid   ErrorOccurredBecauseOf = "token signature is invalid"
	ErrTokenMalformed          ErrorOccurredBecauseOf = "token is malformed"
	ErrTokenWrongAlgorithm     ErrorOccurredBecauseOf = "unexpected signing algorithm"
	ErrTokenWrongIssuer        ErrorOccurredBecauseOf = "token issuer does not match"
	ErrTokenWrongAudience      ErrorOccurredBecauseOf = "token audience does not match"
	ErrMissingSubIdentity      ErrorOccurredBecauseOf = "missing or invalid sub Identity"
	ErrMissingClientIDIdentity ErrorOccurredBecauseOf = "missing or invalid client_id Identity"
	ErrInvalidIdentityType     ErrorOccurredBecauseOf = "unexpected Identities type in token"
	ErrSecretKeyTooShort       ErrorOccurredBecauseOf = "secret key must be at least 32 bytes"
	ErrEmptyIssuer             ErrorOccurredBecauseOf = "issuer must not be empty"
	ErrEmptyAudience           ErrorOccurredBecauseOf = "audience must not be empty"
	ErrNoIdentityInContext     ErrorOccurredBecauseOf = "no identity in request context"
)

// ValidationError provides structured error information for auth failures.
type ValidationError struct {
	TypeOfError            internal.TypeOfError
	ErrorOccurredBecauseOf ErrorOccurredBecauseOf
	Identity               string // e.g. "sub", "client_id", "iss", "aud" — empty otherwise
	Err                    error
}

func (e ValidationError) Error() string {
	if e.Identity != "" {
		return fmt.Sprintf("Identity '%s': %s; cause: %s",
			e.Identity, e.ErrorOccurredBecauseOf, e.Err)
	}
	
	return fmt.Sprintf("auth: %s; cause: %s", e.ErrorOccurredBecauseOf, e.Err)
}

func (e ValidationError) Unwrap() error { return e.Err }

// wrapJWTError maps errors from golang-jwt/jwt into ValidationError.
// This keeps the jwt library as an implementation detail of the auth package —
// callers work with ValidationError and never import golang-jwt directly.
func wrapJWTError(err error) ValidationError {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return ValidationError{ErrTypeTokenExpired, ErrTokenExpired, "", err}
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return ValidationError{ErrTypeTokenNotYetValid, ErrTokenNotYetValid, "", err}
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return ValidationError{ErrTypeTokenInvalid, ErrTokenSignatureInvalid, "", err}
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return ValidationError{ErrTypeTokenWrongIssuer, ErrTokenWrongIssuer, "iss", err}
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return ValidationError{ErrTypeTokenWrongAudience, ErrTokenWrongAudience, "aud", err}
	case errors.Is(err, jwt.ErrTokenUnverifiable):
		return ValidationError{ErrTypeTokenInvalid, ErrTokenWrongAlgorithm, "", err}
	default:
		return ValidationError{ErrTypeTokenInvalid, ErrTokenMalformed, "", err}
	}
}
