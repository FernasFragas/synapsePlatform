package sqllite

import "synapsePlatform/internal/ingestor"

// StoreError marks a SQLite failure as retryable. It satisfies ingestor's
// (unexported) classified interface structurally -- ErrorClass is exported and
// returns the exported ingestor.ErrorClass, so no import cycle is needed.
type StoreError struct {
	Err error
}

func (e *StoreError) Error() string {
	return e.Err.Error()
}
func (e *StoreError) Unwrap() error {
	return e.Err
}
func (e *StoreError) ErrorClass() ingestor.ErrorClass {
	return ingestor.ClassTransient
}
