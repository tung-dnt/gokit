package validator

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps go-playground/validator for use with Echo's Validator interface.
type CustomValidator struct {
	validator *validator.Validate
}

// New returns a CustomValidator backed by a fresh go-playground/validator instance.
// Field names in validation errors use JSON tag names for API-friendly responses.
func New() *CustomValidator {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return ""
		}
		return name
	})
	return &CustomValidator{validator: v}
}

// Validate runs struct-level validation and returns a *ValidationError (HTTP 422) on failure
// with field-level error details (field name → failed constraint tag).
func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			fields := make(map[string]string, len(ve))
			for _, fe := range ve {
				fields[fe.Field()] = fe.Tag()
			}
			return &ValidationError{fields: fields}
		}
		return &ValidationError{fields: map[string]string{"_error": err.Error()}}
	}
	return nil
}

// ValidationError holds field-level validation failures.
// It implements error, echo.HTTPStatusCoder, and json.Marshaler so Echo v5's
// default error handler serialises it as structured JSON with status 422.
type ValidationError struct {
	fields map[string]string
}

// Error implements the error interface.
func (e *ValidationError) Error() string { return "validation failed" }

// StatusCode implements echo.HTTPStatusCoder to produce HTTP 422.
func (e *ValidationError) StatusCode() int { return http.StatusUnprocessableEntity }

// MarshalJSON implements json.Marshaler for structured error output.
func (e *ValidationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"errors": e.fields})
}

// Fields returns the validation error details.
func (e *ValidationError) Fields() map[string]string { return e.fields }
