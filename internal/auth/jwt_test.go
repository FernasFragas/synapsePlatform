package auth_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"synapsePlatform/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

const (
	testIssuer   = "https://auth.example.com"
	testAudience = "synapse-platform-api"
)

var testKey = make([]byte, 32) // 32 zero bytes — valid length for tests

type JWTValidatorTestSuite struct {
	suite.Suite
	validator *auth.JWTValidator
}

func TestJWTValidatorSuite(t *testing.T) {
	suite.Run(t, new(JWTValidatorTestSuite))
}

func (s *JWTValidatorTestSuite) SetupTest() {
	v, err := auth.NewJWTValidator(testKey, testIssuer, testAudience)
	s.Require().NoError(err)
	s.validator = v
}

func (s *JWTValidatorTestSuite) TestNewJWTValidator_ShortKey_ReturnsConfigurationError() {
	_, err := auth.NewJWTValidator([]byte("short"), testIssuer, testAudience)

	s.Require().Error(err)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeConfiguration, valErr.TypeOfError)
	s.Equal(auth.ErrSecretKeyTooShort, valErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestNewJWTValidator_EmptyIssuer_ReturnsConfigurationError() {
	_, err := auth.NewJWTValidator(testKey, "", testAudience)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeConfiguration, valErr.TypeOfError)
	s.Equal(auth.ErrEmptyIssuer, valErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestNewJWTValidator_EmptyAudience_ReturnsConfigurationError() {
	_, err := auth.NewJWTValidator(testKey, testIssuer, "")

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeConfiguration, valErr.TypeOfError)
	s.Equal(auth.ErrEmptyAudience, valErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestNewJWTValidator_ValidParams_ReturnsNoError() {
	_, err := auth.NewJWTValidator(testKey, testIssuer, testAudience)
	s.NoError(err)
}

func (s *JWTValidatorTestSuite) TestValidate_ValidToken_ReturnsIdentity() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	identity, err := s.validator.Validate(token)

	s.Require().NoError(err)
	s.Equal("user-123", identity.Subject)
	s.Equal("my-service", identity.ClientID)
}

func (s *JWTValidatorTestSuite) TestValidate_TokenWithScopes_PopulatesScopes() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"scope":     "read:events write:events",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	identity, err := s.validator.Validate(token)

	s.Require().NoError(err)
	s.Equal([]string{"read:events", "write:events"}, identity.Scopes)
}

func (s *JWTValidatorTestSuite) TestValidate_TokenWithNoScope_ReturnNilScopes() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	identity, err := s.validator.Validate(token)

	s.Require().NoError(err)
	s.Nil(identity.Scopes)
}

func (s *JWTValidatorTestSuite) TestValidate_ExpiredToken_ReturnsTokenExpiredError() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(-time.Hour).Unix(), // in the past
	})

	_, err := s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeTokenExpired, valErr.TypeOfError)
	s.Equal(auth.ErrTokenExpired, valErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestValidate_WrongIssuer_ReturnsWrongIssuerError() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       "https://other-auth.example.com", // wrong issuer
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeTokenWrongIssuer, valErr.TypeOfError)
}

func (s *JWTValidatorTestSuite) TestValidate_WrongAudience_ReturnsWrongAudienceError() {
	token := s.sign(jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{"other-api"}, // wrong audience
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeTokenWrongAudience, valErr.TypeOfError)
}

func (s *JWTValidatorTestSuite) TestValidate_InvalidSignature_ReturnsTokenInvalidError() {
	// Sign with a different key
	otherKey := make([]byte, 32)
	otherKey[0] = 1 // differs from testKey
	otherValidator, _ := auth.NewJWTValidator(otherKey, testIssuer, testAudience)

	token := s.signWith(otherKey, jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := otherValidator.Validate(token)
	// Verify with the original validator — wrong key
	_, err = s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeTokenInvalid, valErr.TypeOfError)
	s.Equal(auth.ErrTokenSignatureInvalid, valErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestValidate_WrongAlgorithm_ReturnsTokenInvalidError() {
	// Sign with ECDSA (ES256) — validator expects HMAC
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)

	t := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":       "user-123",
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	token, err := t.SignedString(ecKey)
	s.Require().NoError(err)

	_, valErr := s.validator.Validate(token)

	var authErr auth.ValidationError
	s.Require().ErrorAs(valErr, &authErr)
	s.Equal(auth.ErrTypeTokenInvalid, authErr.TypeOfError)
	s.Equal(auth.ErrTokenWrongAlgorithm, authErr.ErrorOccurredBecauseOf)
}

func (s *JWTValidatorTestSuite) TestValidate_MalformedToken_ReturnsTokenInvalidError() {
	_, err := s.validator.Validate("not.a.jwt")

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeTokenInvalid, valErr.TypeOfError)
}

func (s *JWTValidatorTestSuite) TestValidate_MissingSubClaim_ReturnsMissingIdentityError() {
	token := s.sign(jwt.MapClaims{
		// "sub" deliberately omitted
		"client_id": "my-service",
		"iss":       testIssuer,
		"aud":       jwt.ClaimStrings{testAudience},
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeMissingIdentity, valErr.TypeOfError)
	s.Equal(auth.ErrMissingSubIdentity, valErr.ErrorOccurredBecauseOf)
	s.Equal("sub", valErr.Identity)
}

func (s *JWTValidatorTestSuite) TestValidate_MissingClientIDClaim_ReturnsMissingIdentityError() {
	token := s.sign(jwt.MapClaims{
		"sub": "user-123",
		// "client_id" deliberately omitted
		"iss": testIssuer,
		"aud": jwt.ClaimStrings{testAudience},
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := s.validator.Validate(token)

	var valErr auth.ValidationError
	s.Require().ErrorAs(err, &valErr)
	s.Equal(auth.ErrTypeMissingIdentity, valErr.TypeOfError)
	s.Equal(auth.ErrMissingClientIDIdentity, valErr.ErrorOccurredBecauseOf)
	s.Equal("client_id", valErr.Identity)
}

func (s *JWTValidatorTestSuite) sign(claims jwt.MapClaims) string {
	return s.signWith(testKey, claims)
}

func (s *JWTValidatorTestSuite) signWith(key []byte, claims jwt.MapClaims) string {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(key)
	s.Require().NoError(err)
	return signed
}
