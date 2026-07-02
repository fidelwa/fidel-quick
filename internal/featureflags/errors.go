package featureflags

import (
	"errors"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// isNotFound reports whether err is a "not_found" AppError, letting the service
// distinguish a missing flag from a real failure.
func isNotFound(err error) bool {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "not_found"
	}
	return false
}
