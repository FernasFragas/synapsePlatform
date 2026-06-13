package sqllite

import (
	"database/sql"
	"errors"
	"synapsePlatform/internal/ingestor"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func classifyStoreError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, sql.ErrTxDone) {
		return ingestor.NewTransientError(err)
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		// modernc.org/sqlite can return extended SQLite codes like
		// SQLITE_BUSY_TIMEOUT. Mask to the primary result code before matching.
		switch primarySQLiteCode(sqliteErr.Code()) {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED, sqlite3.SQLITE_IOERR:
			return ingestor.NewTransientError(err)
		case sqlite3.SQLITE_CONSTRAINT:
			return ingestor.NewTerminalError(err)
		}
	}

	return ingestor.NewTransientError(err)
}

func primarySQLiteCode(code int) int {
	return code & 0xff
}
