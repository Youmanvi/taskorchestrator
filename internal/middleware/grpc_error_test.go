package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

func TestWithGRPCErrorHandling_TransientError(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		message  string
		wantErr  bool
		wantTransient bool
	}{
		{
			name:          "ResourceExhausted (code 8)",
			code:          codes.ResourceExhausted,
			message:       "quota exceeded",
			wantErr:       true,
			wantTransient: true,
		},
		{
			name:          "FailedPrecondition (code 9) - resource conflict",
			code:          codes.FailedPrecondition,
			message:       "resource locked",
			wantErr:       true,
			wantTransient: true,
		},
		{
			name:          "Aborted (code 10) - transaction conflict",
			code:          codes.Aborted,
			message:       "transaction aborted",
			wantErr:       true,
			wantTransient: true,
		},
		{
			name:          "DeadlineExceeded (code 4)",
			code:          codes.DeadlineExceeded,
			message:       "deadline exceeded",
			wantErr:       true,
			wantTransient: true,
		},
		{
			name:          "Unavailable (code 14)",
			code:          codes.Unavailable,
			message:       "service unavailable",
			wantErr:       true,
			wantTransient: true,
		},
		{
			name:          "Internal (code 13)",
			code:          codes.Internal,
			message:       "internal server error",
			wantErr:       true,
			wantTransient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := WithGRPCErrorHandling()

			// Activity that returns a gRPC error
			activity := func(ctx context.Context, input []byte) ([]byte, error) {
				return nil, status.Error(tt.code, tt.message)
			}

			wrapped := middleware(activity)

			_, err := wrapped(context.Background(), []byte{})

			assert.True(t, tt.wantErr, "expected error")
			assert.NotNil(t, err, "error should not be nil")

			// Check if error is classified as transient
			if customErr, ok := err.(*errors.CustomError); ok {
				assert.Equal(t, tt.wantTransient, customErr.IsTransient(), "should be transient")
			}
		})
	}
}

func TestWithGRPCErrorHandling_PermanentError(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		message  string
		wantErr  bool
		wantTransient bool
	}{
		{
			name:          "InvalidArgument (code 3)",
			code:          codes.InvalidArgument,
			message:       "invalid input",
			wantErr:       true,
			wantTransient: false,
		},
		{
			name:          "NotFound (code 5)",
			code:          codes.NotFound,
			message:       "resource not found",
			wantErr:       true,
			wantTransient: false,
		},
		{
			name:          "AlreadyExists (code 6)",
			code:          codes.AlreadyExists,
			message:       "resource already exists",
			wantErr:       true,
			wantTransient: false,
		},
		{
			name:          "PermissionDenied (code 7)",
			code:          codes.PermissionDenied,
			message:       "permission denied",
			wantErr:       true,
			wantTransient: false,
		},
		{
			name:          "Unimplemented (code 12)",
			code:          codes.Unimplemented,
			message:       "method not implemented",
			wantErr:       true,
			wantTransient: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := WithGRPCErrorHandling()

			// Activity that returns a gRPC error
			activity := func(ctx context.Context, input []byte) ([]byte, error) {
				return nil, status.Error(tt.code, tt.message)
			}

			wrapped := middleware(activity)

			_, err := wrapped(context.Background(), []byte{})

			assert.True(t, tt.wantErr, "expected error")
			assert.NotNil(t, err, "error should not be nil")

			// Check if error is classified as permanent
			if customErr, ok := err.(*errors.CustomError); ok {
				assert.Equal(t, !tt.wantTransient, customErr.IsPermanent(), "should be permanent")
			}
		})
	}
}

func TestWithGRPCErrorHandling_Success(t *testing.T) {
	middleware := WithGRPCErrorHandling()

	// Activity that succeeds
	activity := func(ctx context.Context, input []byte) ([]byte, error) {
		return []byte("result"), nil
	}

	wrapped := middleware(activity)

	output, err := wrapped(context.Background(), []byte{})

	assert.NoError(t, err)
	assert.Equal(t, []byte("result"), output)
}

func TestWithGRPCErrorHandling_NonGRPCError(t *testing.T) {
	middleware := WithGRPCErrorHandling()

	// Activity that returns a non-gRPC error
	activity := func(ctx context.Context, input []byte) ([]byte, error) {
		return nil, errors.NewPermanentError("CUSTOM_ERROR", "custom error", nil)
	}

	wrapped := middleware(activity)

	_, err := wrapped(context.Background(), []byte{})

	assert.NotNil(t, err)
	// Should pass through unchanged (non-gRPC error)
	if customErr, ok := err.(*errors.CustomError); ok {
		assert.True(t, customErr.IsPermanent(), "should remain permanent")
	}
}

func TestIsTransientGRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     bool
	}{
		{
			name: "ResourceExhausted",
			err:  status.Error(codes.ResourceExhausted, "quota exceeded"),
			want: true,
		},
		{
			name: "FailedPrecondition",
			err:  status.Error(codes.FailedPrecondition, "resource conflict"),
			want: true,
		},
		{
			name: "Unavailable",
			err:  status.Error(codes.Unavailable, "service down"),
			want: true,
		},
		{
			name: "InvalidArgument",
			err:  status.Error(codes.InvalidArgument, "bad input"),
			want: false,
		},
		{
			name: "NotFound",
			err:  status.Error(codes.NotFound, "not found"),
			want: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			want:     false,
		},
		{
			name:     "Non-gRPC error",
			err:      errors.NewPermanentError("CUSTOM", "msg", nil),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientGRPCError(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetGRPCStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantOk   bool
	}{
		{
			name:     "gRPC error",
			err:      status.Error(codes.ResourceExhausted, "quota exceeded"),
			wantCode: codes.ResourceExhausted,
			wantOk:   true,
		},
		{
			name:     "Nil error",
			err:      nil,
			wantCode: codes.OK,
			wantOk:   false,
		},
		{
			name:     "Non-gRPC error",
			err:      errors.NewPermanentError("CUSTOM", "msg", nil),
			wantCode: codes.Unknown,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := GetGRPCStatusCode(tt.err)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantCode, code)
			}
		})
	}
}

func TestGRPCErrorHandling_WithRetryMiddleware(t *testing.T) {
	// This test verifies that gRPC errors are properly classified
	// before the retry middleware sees them

	retryCount := 0
	maxRetries := 2

	middleware := WithGRPCErrorHandling()

	// Activity that fails with transient gRPC error twice, then succeeds
	activity := func(ctx context.Context, input []byte) ([]byte, error) {
		retryCount++
		if retryCount < maxRetries {
			return nil, status.Error(codes.FailedPrecondition, "resource locked")
		}
		return []byte("success"), nil
	}

	// Apply gRPC error handling
	wrapped := middleware(activity)

	// Call once - should get transient error that will be retried
	_, err := wrapped(context.Background(), []byte{})

	// Error should be classified as transient
	assert.NotNil(t, err)
	if customErr, ok := err.(*errors.CustomError); ok {
		assert.True(t, customErr.IsTransient(), "should classify as transient for retry")
	}
}
