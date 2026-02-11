package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// WithCircuitBreaker returns a middleware that protects activity execution with a circuit breaker
func WithCircuitBreaker(name string, threshold float64, timeout time.Duration) ActivityMiddleware {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    timeout,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= threshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// Log state changes if needed
		},
	})

	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			result, err := cb.Execute(func() (interface{}, error) {
				return next(ctx, input)
			})

			if err != nil {
				// Check if it's a circuit breaker error
				if err == gobreaker.ErrOpenState {
					return nil, errors.NewTransientError(
						"CIRCUIT_BREAKER_OPEN",
						fmt.Sprintf("circuit breaker open for activity: %s", name),
						err,
					)
				}
				return nil, err
			}

			return result.([]byte), nil
		}
	}
}
