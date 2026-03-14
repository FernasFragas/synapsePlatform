//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/auth/mocked_auth.go
package auth

import (
	"context"
	"slices"
)

type Identity struct {
	Subject  string
	ClientID string
	Scopes   []string
}

// TokenValidator — interface defined in the auth domain, consumed by api.
type TokenValidator interface {
	Validate(tokenString string) (Identity, error)
}

type claimsKey struct{}

func WithIdentity(ctx context.Context, c Identity) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

func IdentityFromContext(ctx context.Context) (Identity, error) {
	c, ok := ctx.Value(claimsKey{}).(Identity)
	if !ok {
		return Identity{}, ValidationError{
			TypeOfError:            ErrTypeNoIdentity,
			ErrorOccurredBecauseOf: ErrNoIdentityInContext,
		}
	}

	return c, nil
}

// HasScope reports whether the claims contain the given scope.
func (c Identity) HasScope(scope string) bool {
	return slices.Contains(c.Scopes, scope)
}
