package apperror

import "fmt"

// AppError represents a structured application error with HTTP status mapping.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Cause      error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NotFound(message string, cause error) *AppError {
	return &AppError{Code: "not_found", Message: message, HTTPStatus: 404, Cause: cause}
}

func BadRequest(message string, cause error) *AppError {
	return &AppError{Code: "bad_request", Message: message, HTTPStatus: 400, Cause: cause}
}

func Internal(message string, cause error) *AppError {
	return &AppError{Code: "internal_error", Message: message, HTTPStatus: 500, Cause: cause}
}

func Conflict(message string, cause error) *AppError {
	return &AppError{Code: "conflict", Message: message, HTTPStatus: 409, Cause: cause}
}

func TooManyRequests(message string, cause error) *AppError {
	return &AppError{Code: "too_many_requests", Message: message, HTTPStatus: 429, Cause: cause}
}
