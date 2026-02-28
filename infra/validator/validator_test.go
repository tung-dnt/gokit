package validator

import (
	"errors"
	"net/http"
	"testing"
)

type testInput struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

func TestCustomValidator_Validate(t *testing.T) {
	t.Parallel()

	cv := New()

	tests := []struct {
		name     string
		input    any
		wantErr  bool
		wantCode int
	}{
		{
			name:    "valid struct",
			input:   &testInput{Name: "Alice", Email: "alice@example.com"},
			wantErr: false,
		},
		{
			name:     "invalid struct returns 422",
			input:    &testInput{},
			wantErr:  true,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "partial invalid returns 422",
			input:    &testInput{Name: "Alice"},
			wantErr:  true,
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := cv.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				return
			}

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatal("expected *ValidationError")
			}
			if ve.StatusCode() != tt.wantCode {
				t.Errorf("StatusCode() = %d, want %d", ve.StatusCode(), tt.wantCode)
			}
		})
	}
}

func TestCustomValidator_Validate_FieldDetails(t *testing.T) {
	t.Parallel()

	cv := New()

	err := cv.Validate(&testInput{})
	if err == nil {
		t.Fatal("expected error for empty struct")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("expected *ValidationError")
	}

	fields := ve.Fields()

	// Field names should be JSON tag names (lowercase), not Go field names.
	if _, exists := fields["name"]; !exists {
		t.Errorf("expected error for field 'name', got %v", fields)
	}
	if _, exists := fields["email"]; !exists {
		t.Errorf("expected error for field 'email', got %v", fields)
	}
}

func TestCustomValidator_Validate_MarshalJSON(t *testing.T) {
	t.Parallel()

	cv := New()

	err := cv.Validate(&testInput{})
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("expected *ValidationError")
	}

	b, jsonErr := ve.MarshalJSON()
	if jsonErr != nil {
		t.Fatalf("MarshalJSON() error = %v", jsonErr)
	}
	got := string(b)
	if got == "" {
		t.Error("expected non-empty JSON")
	}
	// Should contain "errors" key with field names.
	for _, key := range []string{`"errors"`, `"name"`, `"email"`} {
		if !contains(got, key) {
			t.Errorf("JSON %q missing expected key %s", got, key)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
