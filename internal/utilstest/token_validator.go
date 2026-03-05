package utilstest

import (
	"testing"

	"synapsePlatform/internal/auth"
	mock_auth "synapsePlatform/internal/utilstest/mocksgen/auth"

	"go.uber.org/mock/gomock"
)

type TokenValidator struct {
	*mock_auth.MockTokenValidator
	t *testing.T
}

func NewTokenValidator(t *testing.T) *TokenValidator {
	return &TokenValidator{
		MockTokenValidator: mock_auth.NewMockTokenValidator(gomock.NewController(t)),
		t:                  t,
	}
}

func (tv *TokenValidator) WithIdentity(identity auth.Identity) *TokenValidator {
	tv.EXPECT().Validate(gomock.Any()).Return(identity, nil)

	return tv
}

func (tv *TokenValidator) WithError(err error) *TokenValidator {
	tv.EXPECT().Validate(gomock.Any()).Return(auth.Identity{}, err)

	return tv
}
