package userhdl

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// assertValidationFields checks that err contains validator.ValidationErrors with the expected fields.
func assertValidationFields(t *testing.T, err error, wantFields []string) {
	t.Helper()

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatal("expected validator.ValidationErrors")
	}

	gotFields := make(map[string]bool, len(ve))
	for _, fe := range ve {
		gotFields[fe.Field()] = true
	}
	for _, want := range wantFields {
		if !gotFields[want] {
			t.Errorf("expected error on field %q, got errors: %v", want, ve)
		}
	}
}

func TestCreateUserRequest(t *testing.T) {
	t.Parallel()

	v := validator.New()

	tests := []struct {
		name      string
		input     CreateUserRequest
		wantErr   bool
		errFields []string
	}{
		{name: "valid input", input: CreateUserRequest{Name: "Alice", Email: "alice@example.com"}},
		{name: "missing name", input: CreateUserRequest{Email: "alice@example.com"}, wantErr: true, errFields: []string{"Name"}},
		{name: "missing email", input: CreateUserRequest{Name: "Alice"}, wantErr: true, errFields: []string{"Email"}},
		{name: "empty name", input: CreateUserRequest{Name: "", Email: "alice@example.com"}, wantErr: true, errFields: []string{"Name"}},
		{name: "name too long", input: CreateUserRequest{Name: strings.Repeat("a", 101), Email: "alice@example.com"}, wantErr: true, errFields: []string{"Name"}},
		{name: "invalid email format", input: CreateUserRequest{Name: "Alice", Email: "not-an-email"}, wantErr: true, errFields: []string{"Email"}},
		{name: "missing both fields", input: CreateUserRequest{}, wantErr: true, errFields: []string{"Name", "Email"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Struct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assertValidationFields(t, err, tt.errFields)
			}
		})
	}
}

func TestUpdateUserRequest(t *testing.T) {
	t.Parallel()

	v := validator.New()

	tests := []struct {
		name      string
		input     UpdateUserRequest
		wantErr   bool
		errFields []string
	}{
		{name: "valid full update", input: UpdateUserRequest{Name: "Bob", Email: "bob@example.com"}},
		{name: "empty struct", input: UpdateUserRequest{}},
		{name: "name only", input: UpdateUserRequest{Name: "Bob"}},
		{name: "email only", input: UpdateUserRequest{Email: "bob@example.com"}},
		{name: "name too long", input: UpdateUserRequest{Name: strings.Repeat("a", 101)}, wantErr: true, errFields: []string{"Name"}},
		{name: "invalid email", input: UpdateUserRequest{Email: "bad-email"}, wantErr: true, errFields: []string{"Email"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Struct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assertValidationFields(t, err, tt.errFields)
			}
		})
	}
}
