package auth

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	issuer    string
	audience  string
	secretKey []byte
}

func NewJWTValidator(secretKey []byte, issuer, audience string) (*JWTValidator, error) {
	if len(secretKey) < 32 {
		return nil, ValidationError{ErrTypeConfiguration, ErrSecretKeyTooShort, "", nil}
	}

	if issuer == "" {
		return nil, ValidationError{ErrTypeConfiguration, ErrEmptyIssuer, "", nil}
	}

	if audience == "" {
		return nil, ValidationError{ErrTypeConfiguration, ErrEmptyAudience, "", nil}
	}

	return &JWTValidator{secretKey: secretKey, issuer: issuer, audience: audience}, nil
}

func (v *JWTValidator) Validate(tokenString string) (Identity, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ValidationError{
				TypeOfError:            ErrTypeTokenInvalid,
				ErrorOccurredBecauseOf: ErrTokenWrongAlgorithm,
				Err:                    fmt.Errorf("got %v", t.Header["alg"]),
			}
		}

		return v.secretKey, nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
	)
	if err != nil || !token.Valid {
		return Identity{}, wrapJWTError(err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Identity{}, ValidationError{
			TypeOfError:            ErrTypeTokenInvalid,
			ErrorOccurredBecauseOf: ErrInvalidIdentityType,
		}
	}

	sub, ok := mapClaims["sub"].(string)
	if !ok || sub == "" {
		return Identity{}, ValidationError{
			TypeOfError:            ErrTypeMissingIdentity,
			ErrorOccurredBecauseOf: ErrMissingSubIdentity,
			Identity:               "sub",
		}
	}

	clientID, ok := mapClaims["client_id"].(string)
	if !ok || clientID == "" {
		return Identity{}, ValidationError{
			TypeOfError:            ErrTypeMissingIdentity,
			ErrorOccurredBecauseOf: ErrMissingClientIDIdentity,
			Identity:               "client_id",
		}
	}

	return Identity{
		Subject:  sub,
		ClientID: clientID,
		Scopes:   parseScopes(mapClaims),
	}, nil
}

func parseScopes(c jwt.MapClaims) []string {
	switch v := c["scope"].(type) {
	case string:
		if v == "" {
			return nil
		}
		return strings.Split(v, " ")

	case []interface{}:
		scopes := make([]string, 0, len(v))

		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				scopes = append(scopes, s)
			}
		}

		if len(scopes) == 0 {
			return nil
		}

		return scopes

	default:
		return nil
	}
}
