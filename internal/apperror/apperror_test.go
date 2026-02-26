package apperror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	cause := errors.New("underlying")
	err := NotFound("resource not found", cause)

	assert.Equal(t, "not_found", err.Code)
	assert.Equal(t, "resource not found", err.Message)
	assert.Equal(t, 404, err.HTTPStatus)
	assert.Equal(t, cause, err.Cause)
	assert.Contains(t, err.Error(), "resource not found")
	assert.Contains(t, err.Error(), "underlying")
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input", nil)

	assert.Equal(t, "bad_request", err.Code)
	assert.Equal(t, 400, err.HTTPStatus)
	assert.Nil(t, err.Cause)
	assert.Equal(t, "invalid input", err.Error())
}

func TestInternal(t *testing.T) {
	cause := errors.New("db error")
	err := Internal("server error", cause)

	assert.Equal(t, "internal_error", err.Code)
	assert.Equal(t, 500, err.HTTPStatus)
	assert.Equal(t, cause, err.Cause)
}

func TestConflict(t *testing.T) {
	err := Conflict("duplicate key", nil)

	assert.Equal(t, "conflict", err.Code)
	assert.Equal(t, 409, err.HTTPStatus)
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Internal("wrapper", cause)

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

func TestAppError_ErrorsAs(t *testing.T) {
	appErr := NotFound("not found", nil)
	var target *AppError

	assert.True(t, errors.As(appErr, &target))
	assert.Equal(t, "not_found", target.Code)
}

func TestAppError_NilCause(t *testing.T) {
	err := NotFound("not found", nil)
	assert.Equal(t, "not found", err.Error())
	assert.Nil(t, err.Unwrap())
}
