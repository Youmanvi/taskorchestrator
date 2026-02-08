package middleware

import (
	"context"
	"math"
	"time"

	"github.com/vihan/taskorchestrator/internal/infrastructure/observability"
	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

// RetryPolicy defines the retry strategy
type RetryPolicy struct {
	MaxAttempts      int           // Maximum number of retry attempts
	InitialBackoff   time.Duration // Initial backoff duration
	MaxBackoff       time.Duration // Maximum backoff duration
	BackoffMultiplier float64      // Exponential backoff multiplier
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy(maxAttempts int) RetryPolicy {
	return RetryPolicy{
		MaxAttempts:      maxAttempts,
		InitialBackoff:   100 * time.Millisecond,
		MaxBackoff:       30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// WithRetry returns a middleware that retries the activity on transient failures
func WithRetry(logger *observability.Logger, policy RetryPolicy) ActivityMiddleware {
	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			var lastErr error
			var attempt int

			for attempt = 1; attempt <= policy.MaxAttempts; attempt++ {
				result, err := next(ctx, input)

				if err == nil {
					return result, nil
				}

				// Check if error is transient
				if customErr, ok := err.(*errors.CustomError); ok && !customErr.IsTransient() {
					// Permanent error, don't retry
					return nil, err
				}

				lastErr = err

				if attempt < policy.MaxAttempts {
					backoff := calculateBackoff(attempt-1, policy)
					logger.WithError(err).Debug("retrying activity after backoff")
					select {
					case <-time.After(backoff):
						// Continue to next attempt
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}

			return nil, lastErr
		}
	}
}

// calculateBackoff calculates exponential backoff
func calculateBackoff(attempt int, policy RetryPolicy) time.Duration {
	backoff := float64(policy.InitialBackoff.Milliseconds()) *
		math.Pow(policy.BackoffMultiplier, float64(attempt))

	maxMs := float64(policy.MaxBackoff.Milliseconds())
	if backoff > maxMs {
		backoff = maxMs
	}

	return time.Duration(backoff) * time.Millisecond
}
