package auth_test

import (
	"errors"
	"testing"

	"synapsePlatform/internal/auth"

	"github.com/stretchr/testify/suite"
)

type ValidationErrorTestSuite struct {
	suite.Suite
}

func TestValidationErrorSuite(t *testing.T) {
	suite.Run(t, new(ValidationErrorTestSuite))
}

func (s *ValidationErrorTestSuite) TestValidationError_WithIdentityField_ContainsClaimName() {
	err := auth.ValidationError{
		TypeOfError:            auth.ErrTypeMissingIdentity,
		ErrorOccurredBecauseOf: auth.ErrMissingSubIdentity,
		Identity:               "sub",
		Err:                    errors.New("was nil"),
	}

	msg := err.Error()

	s.Contains(msg, "sub")
	s.Contains(msg, "was nil")
}

func (s *ValidationErrorTestSuite) TestValidationError_WithoutIdentityField_OmitsClaimName() {
	err := auth.ValidationError{
		TypeOfError:            auth.ErrTypeTokenExpired,
		ErrorOccurredBecauseOf: auth.ErrTokenExpired,
	}

	s.NotContains(err.Error(), "Identity '")
}

func (s *ValidationErrorTestSuite) TestValidationError_Unwrap_ExposesUnderlyingCause() {
	cause := errors.New("root cause")
	err := auth.ValidationError{Err: cause}

	s.ErrorIs(err, cause)
}

func (s *ValidationErrorTestSuite) TestValidationError_ErrorsAs_ExtractsThroughWrap() {
	original := auth.ValidationError{
		TypeOfError: auth.ErrTypeTokenExpired,
		Identity:    "exp",
	}
	wrapped := errors.Join(errors.New("outer context"), original)

	var extracted auth.ValidationError
	s.Require().True(errors.As(wrapped, &extracted))
	s.Equal(auth.ErrTypeTokenExpired, extracted.TypeOfError)
	s.Equal("exp", extracted.Identity)
}

func (s *ValidationErrorTestSuite) TestValidationError_ImplementsError() {
	var err error = auth.ValidationError{}
	s.NotNil(err)
}
