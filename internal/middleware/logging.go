package middleware

import (
	"context"
	"time"

	"github.com/vihan/taskorchestrator/internal/infrastructure/observability"
)

// WithLogging returns a middleware that logs activity execution
func WithLogging(logger *observability.Logger, activityName string) ActivityMiddleware {
	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			start := time.Now()
			actLogger := logger.WithActivityName(activityName)

			actLogger.Logger.Info().Msg("activity started")

			output, err := next(ctx, input)

			duration := time.Since(start)

			if err != nil {
				actLogger.WithError(err).Logger.Error().
					Dur("duration_ms", duration).
					Msg("activity failed")
				return nil, err
			}

			actLogger.Logger.Info().
				Dur("duration_ms", duration).
				Msg("activity completed")

			return output, nil
		}
	}
}
