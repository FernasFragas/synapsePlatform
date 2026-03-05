//nolint:depguard
package utilstest


import (
	"context"
	"testing"

	"synapsePlatform/internal/ingestor"
	"synapsePlatform/internal/sqllite"

	"github.com/stretchr/testify/require"
)

type TestRepo struct {
	repo *sqllite.Repo

	t *testing.T
}

// NewTestRepo creates an in-memory SQLite database for testing.
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	repo, err := sqllite.NewRepo(":memory:")
	require.NoError(t, err, "failed to create test database")

	t.Cleanup(func() {
		repo.Close()
	})

	return &TestRepo{
		repo: repo,
		t: t,
	}
}

// SeedEvents seeds the database with test events.
func (tr *TestRepo) SeedEvents(t *testing.T, repo *sqllite.Repo, events []*ingestor.BaseEvent) *TestRepo {
	t.Helper()

	ctx := context.Background()

	for _, event := range events {
		err := repo.StoreData(ctx, event)
		require.NoError(t, err, "failed to insert tr event")
	}

	return tr
}

func (tr *TestRepo) Build() *sqllite.Repo {
	return tr.repo
}
