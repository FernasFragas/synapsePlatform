package auth_test

import (
	"context"
	"testing"

	"synapsePlatform/internal/auth"

	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestWithIdentity_ThenFromContext_ReturnsIdentity() {
	expected := auth.Identity{
		Subject:  "user-123",
		ClientID: "my-service",
		Scopes:   []string{"read:events"},
	}
	ctx := auth.WithIdentity(context.Background(), expected)

	got, err := auth.IdentityFromContext(ctx)

	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *AuthTestSuite) TestIdentityFromContext_EmptyContext_ReturnsValidationError() {
	_, err := auth.IdentityFromContext(context.Background())

	s.Require().Error(err)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeNoIdentity, valErr.TypeOfError)
	s.Equal(auth.ErrNoIdentityInContext, valErr.ErrorOccurredBecauseOf)
}

func (s *AuthTestSuite) TestIdentityFromContext_WrongType_ReturnsValidationError() {
	// Pollute context with a string under the same key shape
	ctx := context.WithValue(context.Background(), struct{}{}, "not-an-identity")

	_, err := auth.IdentityFromContext(ctx)

	s.Require().Error(err)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeNoIdentity, valErr.TypeOfError)
}

func (s *AuthTestSuite) TestHasScope_MatchingScope_ReturnsTrue() {
	identity := auth.Identity{Scopes: []string{"read:events", "write:events"}}
	s.True(identity.HasScope("read:events"))
}

func (s *AuthTestSuite) TestHasScope_NoMatchingScope_ReturnsFalse() {
	identity := auth.Identity{Scopes: []string{"write:events"}}
	s.False(identity.HasScope("read:events"))
}

func (s *AuthTestSuite) TestHasScope_EmptyScopes_ReturnsFalse() {
	identity := auth.Identity{Scopes: nil}
	s.False(identity.HasScope("read:events"))
}

func (s *AuthTestSuite) TestHasScope_ExactMatch_RequiredNotSubstring() {
	// "read" must not match "read:events"
	identity := auth.Identity{Scopes: []string{"read:events"}}
	s.False(identity.HasScope("read"))
}
