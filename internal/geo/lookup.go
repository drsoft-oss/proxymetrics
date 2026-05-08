package geo

import (
	"context"
	"errors"
	"net/http"
)

// Lookup resolves the exit IP and its geo / connection-type for the given client.
// The client's transport is expected to dial through the proxy under test (or
// directly, depending on the implementation).
type Lookup interface {
	Lookup(ctx context.Context, client *http.Client) (Result, error)
}

// Cascade tries each Lookup in order, returning the first non-error result.
// If all fail, it returns the last error wrapped with the count of attempts.
type Cascade []Lookup

// Lookup runs the cascade.
func (c Cascade) Lookup(ctx context.Context, client *http.Client) (Result, error) {
	if len(c) == 0 {
		return Result{}, errors.New("geo: empty cascade")
	}
	var lastErr error
	for _, l := range c {
		r, err := l.Lookup(ctx, client)
		if err == nil {
			return r, nil
		}
		lastErr = err
	}
	return Result{}, lastErr
}
