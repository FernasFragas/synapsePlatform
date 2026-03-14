package sqllite

import (
	"context"
	"encoding/json"
	"synapsePlatform/internal/ingestor"
)

type FailureStorer struct {
	repo *Repo
}

func NewFailureStorer(repo *Repo) *FailureStorer {
	return &FailureStorer{repo: repo}
}

func (s *FailureStorer) StoreFailure(ctx context.Context, failed ingestor.FailedMessage) error {
	var msgJSON []byte
	if failed.Message != nil {
		msgJSON, _ = json.Marshal(failed.Message)
	}

	var errText string
	if failed.Err != nil {
		errText = failed.Err.Error()
	}

	_, err := s.repo.Db.ExecContext(ctx,
		`INSERT INTO failed_messages (stage, message, error, created_at) VALUES (?, ?, ?, datetime('now'))`,
		failed.Stage, string(msgJSON), errText,
	)

	return err
}