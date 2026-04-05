package router

import (
	"encoding/json"
	"net/http"

	"restful-boilerplate/internal/app"
)

// WriteJSON encodes v as JSON and writes it to w with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck,gosec // best-effort write to response
}

// Bind decodes r.Body into v and validates it. Returns false and writes the error response if either fails.
func Bind(val app.Validator, w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return false
	}
	if err := val.Validate(value); err != nil {
		WriteJSON(w, http.StatusUnprocessableEntity, err)
		return false
	}
	return true
}
