package middleware

import (
	"context"
	"time"

	"github.com/vihan/taskorchestrator/internal/infrastructure/observability"
)

// WithLogging returns a middleware that logs activity execution with persistence
func WithLogging(logger *observability.Logger, activityName string) ActivityMiddleware {
	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			start := time.Now()

			// Extract trace ID from context or generate new one
			traceID := extractTraceID(ctx)
			if traceID == "" {
				traceID = generateTraceID()
			}

			// Add trace context to logger
			actLogger := logger.WithTraceID(ctx, traceID).WithActivityName(activityName)

			// Log start
			actLogger.Logger.Debug().Msg("activity started")

			// Write to repository if configured
			startRecord := observability.NewLogRecord(observability.LogLevelDebug, traceID, "activity started").
				WithActivity(activityName).
				WithInput(input)
			logger.WriteLogRecord(startRecord)

			// Execute activity
			output, err := next(ctx, input)

			duration := time.Since(start)

			// Log result
			if err != nil {
				actLogger.WithError(err).Logger.Error().
					Dur("duration_ms", duration).
					Msg("activity failed")

				// Write error to repository
				errRecord := observability.NewLogRecord(observability.LogLevelError, traceID, "activity failed").
					WithActivity(activityName).
					WithDuration(duration).
					WithInput(input).
					WithError(err.Error())
				logger.WriteLogRecord(errRecord)

				return nil, err
			}

			actLogger.Logger.Info().
				Dur("duration_ms", duration).
				Msg("activity completed")

			// Write completion to repository
			completeRecord := observability.NewLogRecord(observability.LogLevelInfo, traceID, "activity completed").
				WithActivity(activityName).
				WithDuration(duration).
				WithInput(input).
				WithOutput(output)
			logger.WriteLogRecord(completeRecord)

			return output, nil
		}
	}
}

// extractTraceID extracts trace ID from context
func extractTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

// generateTraceID generates a new cryptographic trace ID
func generateTraceID() string {
	id, _ := observability.GenerateCryptographicTraceID()
	return id
}
