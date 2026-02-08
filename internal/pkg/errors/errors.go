package errors

import (
	"fmt"
)

// ErrorType represents the classification of an error
type ErrorType int

const (
	// ErrorTypeTransient indicates a temporary failure that can be retried
	ErrorTypeTransient ErrorType = iota
	// ErrorTypePermanent indicates a permanent failure that should not be retried
	ErrorTypePermanent
	// ErrorTypeTimeout indicates a timeout error
	ErrorTypeTimeout
)

// CustomError is a custom error with classification and context
type CustomError struct {
	Type    ErrorType
	Message string
	Cause   error
	Code    string
}

// NewTransientError creates a new transient error
func NewTransientError(code, message string, cause error) *CustomError {
	return &CustomError{
		Type:    ErrorTypeTransient,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewPermanentError creates a new permanent error
func NewPermanentError(code, message string, cause error) *CustomError {
	return &CustomError{
		Type:    ErrorTypePermanent,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(code, message string) *CustomError {
	return &CustomError{
		Type:    ErrorTypeTimeout,
		Code:    code,
		Message: message,
	}
}

// Error implements the error interface
func (e *CustomError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *CustomError) Unwrap() error {
	return e.Cause
}

// IsTransient returns true if the error is transient
func (e *CustomError) IsTransient() bool {
	return e.Type == ErrorTypeTransient
}

// IsPermanent returns true if the error is permanent
func (e *CustomError) IsPermanent() bool {
	return e.Type == ErrorTypePermanent
}

// IsTimeout returns true if the error is a timeout
func (e *CustomError) IsTimeout() bool {
	return e.Type == ErrorTypeTimeout
}

// ClassifyError attempts to classify a regular error
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypePermanent
	}

	if customErr, ok := err.(*CustomError); ok {
		return customErr.Type
	}

	// Default to permanent for unknown errors
	return ErrorTypePermanent
}
