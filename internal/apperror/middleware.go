package apperror

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandler returns Gin middleware that converts AppError instances
// attached via c.Error() into proper JSON responses with correct HTTP status.
func ErrorHandler(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		var appErr *AppError
		if errors.As(err, &appErr) {
			if appErr.HTTPStatus >= 500 {
				log.Error("internal error",
					"code", appErr.Code,
					"message", appErr.Message,
					"cause", appErr.Cause,
					"path", c.Request.URL.Path,
				)
				c.JSON(appErr.HTTPStatus, gin.H{"error": "error interno", "code": appErr.Code})
			} else {
				c.JSON(appErr.HTTPStatus, gin.H{"error": appErr.Message, "code": appErr.Code})
			}
			return
		}

		// Unknown error — treat as 500, don't expose details
		log.Error("unhandled error",
			"error", err,
			"path", c.Request.URL.Path,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno", "code": "internal_error"})
	}
}
