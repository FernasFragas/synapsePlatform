package log

import (
	"log/slog"
	"synapsePlatform/internal/auth"
)

type AuthLogger struct {
	logger    *slog.Logger
	validator auth.TokenValidator
}

func NewAuthLogger(logger *slog.Logger, validator auth.TokenValidator) *AuthLogger {
	return &AuthLogger{logger: logger, validator: validator}
}

func (al *AuthLogger) Validate(tokenString string) (auth.Identity, error) {
	identity, err := al.validator.Validate(tokenString)
	if err != nil {
		al.logger.Error("failed to validate token", "error", err)

		return auth.Identity{}, err
	}

	al.logger.Info("token validated", "subject", identity.Subject, "client_id", identity.ClientID)

	return identity, nil
}
