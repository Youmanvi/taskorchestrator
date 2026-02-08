package middleware

import (
	"context"
	"time"

	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

// WithTimeout returns a middleware that enforces a timeout on activity execution
func WithTimeout(timeout time.Duration) ActivityMiddleware {
	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			// Create a context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Use a channel to handle the result
			type result struct {
				output []byte
				err    error
			}
			resultChan := make(chan result, 1)

			go func() {
				output, err := next(timeoutCtx, input)
				resultChan <- result{output, err}
			}()

			select {
			case res := <-resultChan:
				return res.output, res.err
			case <-timeoutCtx.Done():
				return nil, errors.NewTimeoutError("ACTIVITY_TIMEOUT", "activity execution exceeded timeout")
			}
		}
	}
}
