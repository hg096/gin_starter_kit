package websocket

import (
	stderrors "errors"
	appErrors "gin_starter/pkg/errors"
	"testing"
)

func TestIsRetryableDBConnError(t *testing.T) {
	retryable := []error{
		stderrors.New("driver: bad connection"),
		stderrors.New("read tcp: connection reset by peer"),
		appErrors.Wrap(stderrors.New("invalid connection"), "DATABASE_ERROR", "wrapped db error"),
	}
	for _, err := range retryable {
		if !isRetryableDBConnError(err) {
			t.Fatalf("expected retryable error, got: %v", err)
		}
	}

	nonRetryable := []error{
		stderrors.New("duplicate entry"),
		appErrors.New("BAD_REQUEST", "invalid payload"),
	}
	for _, err := range nonRetryable {
		if isRetryableDBConnError(err) {
			t.Fatalf("expected non-retryable error, got: %v", err)
		}
	}
}
