package validator

import (
	"github.com/go-playground/validator/v10" // installed by `go get github.com/go-playground/validator/v10`
	"github.com/labstack/echo/v5"
)

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}
