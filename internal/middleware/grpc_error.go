package middleware

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// gRPC status codes that should be treated as transient/retryable
var transientGRPCCodes = map[codes.Code]bool{
	codes.Unavailable:        true,  // 14 - Service temporarily unavailable
	codes.ResourceExhausted:   true,  // 8 - Resource exhausted (quota, rate limits)
	codes.FailedPrecondition:  true,  // 9 - Precondition failed (resource conflicts, state issues)
	codes.Aborted:             true,  // 10 - Request aborted (transaction conflicts)
	codes.DeadlineExceeded:    true,  // 4 - Request deadline exceeded
	codes.Internal:            true,  // 13 - Internal server error (transient)
	codes.Unavailable:         true,  // 14 - Service unavailable
	codes.Unknown:             true,  // 2 - Unknown errors (might be transient)
}

// WithGRPCErrorHandling returns middleware that classifies gRPC errors as transient
// when appropriate, enabling automatic retries for resource conflicts
func WithGRPCErrorHandling() ActivityMiddleware {
	return func(next ActivityFunc) ActivityFunc {
		return func(ctx context.Context, input []byte) ([]byte, error) {
			output, err := next(ctx, input)

			if err != nil {
				// Check if this is a gRPC error
				st, ok := status.FromError(err)
				if ok {
					code := st.Code()

					// If it's a transient gRPC error, convert to transient error for retry
					if transientGRPCCodes[code] {
						return nil, errors.NewTransientError(
							fmt.Sprintf("GRPC_%s", code.String()),
							fmt.Sprintf("gRPC error (transient): %s", st.Message()),
							err,
						)
					}

					// For other gRPC errors, treat as permanent
					return nil, errors.NewPermanentError(
						fmt.Sprintf("GRPC_%s", code.String()),
						fmt.Sprintf("gRPC error (permanent): %s", st.Message()),
						err,
					)
				}
			}

			return output, err
		}
	}
}

// IsTransientGRPCError checks if an error is a gRPC error with a transient status code
func IsTransientGRPCError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return transientGRPCCodes[st.Code()]
}

// GetGRPCStatusCode extracts the gRPC status code from an error, if present
func GetGRPCStatusCode(err error) (codes.Code, bool) {
	if err == nil {
		return codes.OK, false
	}

	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown, false
	}

	return st.Code(), true
}
