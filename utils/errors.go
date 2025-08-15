package utils

import (
	"fmt"
	"net/http"
	"time"

	"gemini-antiblock/logger"
)

// ErrorType represents different types of errors for better handling
type ErrorType int

const (
	ErrorTypeTemporary ErrorType = iota
	ErrorTypePermanent
	ErrorTypeRateLimit
	ErrorTypeAuth
	ErrorTypeNetwork
	ErrorTypeMemory
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Type        ErrorType
	StatusCode  int
	Message     string
	RetryAfter  time.Duration
	Retryable   bool
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("Error (Type: %d, Status: %d): %s", e.Type, e.StatusCode, e.Message)
}

// ClassifyError classifies an HTTP error for better retry handling
func ClassifyError(statusCode int, err error) *RetryableError {
	retryableError := &RetryableError{
		StatusCode: statusCode,
		Message:    "Unknown error",
		Retryable:  false,
	}

	if err != nil {
		retryableError.Message = err.Error()
	}

	switch statusCode {
	case 400, 404:
		// Client errors - not retryable
		retryableError.Type = ErrorTypePermanent
		retryableError.Retryable = false
		retryableError.Message = "Client error - request invalid"

	case 401, 403:
		// Authentication/Authorization errors - not retryable
		retryableError.Type = ErrorTypeAuth
		retryableError.Retryable = false
		retryableError.Message = "Authentication/Authorization error"

	case 413:
		// Request too large - not retryable
		retryableError.Type = ErrorTypeMemory
		retryableError.Retryable = false
		retryableError.Message = "Request too large"

	case 429:
		// Rate limit - retryable with delay
		retryableError.Type = ErrorTypeRateLimit
		retryableError.Retryable = true
		retryableError.RetryAfter = 5 * time.Second // Default retry after 5 seconds
		retryableError.Message = "Rate limit exceeded"

	case 500, 502, 503:
		// Server errors - retryable
		retryableError.Type = ErrorTypeTemporary
		retryableError.Retryable = true
		retryableError.RetryAfter = 1 * time.Second
		retryableError.Message = "Server error - temporary"

	case 504:
		// Gateway timeout - retryable
		retryableError.Type = ErrorTypeNetwork
		retryableError.Retryable = true
		retryableError.RetryAfter = 2 * time.Second
		retryableError.Message = "Gateway timeout"

	default:
		if statusCode >= 500 {
			// Other server errors - retryable
			retryableError.Type = ErrorTypeTemporary
			retryableError.Retryable = true
			retryableError.RetryAfter = 1 * time.Second
		} else {
			// Other client errors - not retryable
			retryableError.Type = ErrorTypePermanent
			retryableError.Retryable = false
		}
	}

	logger.LogDebug(fmt.Sprintf("Classified error: Status=%d, Type=%d, Retryable=%t, RetryAfter=%v",
		statusCode, retryableError.Type, retryableError.Retryable, retryableError.RetryAfter))

	return retryableError
}

// CalculateBackoffDelay calculates exponential backoff delay
func CalculateBackoffDelay(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	// Exponential backoff: baseDelay * 2^attempt
	delay := baseDelay
	for i := 0; i < attempt && delay < maxDelay/2; i++ {
		delay *= 2
	}

	if delay > maxDelay {
		delay = maxDelay
	}

	logger.LogDebug(fmt.Sprintf("Calculated backoff delay for attempt %d: %v", attempt, delay))
	return delay
}

// IsRetryableHTTPError checks if an HTTP status code represents a retryable error
func IsRetryableHTTPError(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return statusCode >= 500
	}
}

// GetRetryDelay returns appropriate retry delay for different error types
func GetRetryDelay(errorType ErrorType, attempt int, baseDelay time.Duration) time.Duration {
	switch errorType {
	case ErrorTypeRateLimit:
		return CalculateBackoffDelay(attempt, 5*time.Second, 60*time.Second)
	case ErrorTypeNetwork:
		return CalculateBackoffDelay(attempt, 2*time.Second, 30*time.Second)
	case ErrorTypeTemporary:
		return CalculateBackoffDelay(attempt, baseDelay, 10*time.Second)
	default:
		return baseDelay
	}
}
