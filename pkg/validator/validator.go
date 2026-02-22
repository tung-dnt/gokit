package validator

import (
	"github.com/go-playground/validator/v10" // installed by `go get github.com/go-playground/validator/v10`
	"github.com/labstack/echo/v5"
)

// CustomValidator wraps go-playground/validator for use with Echo's Validator interface.
type CustomValidator struct {
	validator *validator.Validate
}

// New returns a CustomValidator backed by a fresh go-playground/validator instance.
func New() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

// Validate runs struct-level validation and returns an echo.ErrBadRequest on failure.
func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}
