package middleware

import "context"

// ActivityFunc is the signature of an activity function
type ActivityFunc func(ctx context.Context, input []byte) ([]byte, error)

// ActivityMiddleware is a function that wraps an ActivityFunc
type ActivityMiddleware func(ActivityFunc) ActivityFunc

// ApplyMiddleware applies a chain of middleware to an activity function
func ApplyMiddleware(activity ActivityFunc, middlewares ...ActivityMiddleware) ActivityFunc {
	// Apply middleware in reverse order so they wrap correctly
	for i := len(middlewares) - 1; i >= 0; i-- {
		activity = middlewares[i](activity)
	}
	return activity
}
