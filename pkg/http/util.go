package router

import (
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/internal/app"
	"restful-boilerplate/pkg/telemetry"
)

// WriteJSON encodes v as JSON and writes it to w with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck,gosec // best-effort write to response
}

// Bind decodes r.Body into v and validates it. Returns false and writes the
// error response if either fails. Tags the active OTel span with a
// low-cardinality error.type attribute so validation vs malformed-JSON
// failures are queryable without polluting the server span's error status
// (4xx outcomes stay Unset per OTel HTTP semconv).
func Bind(val app.Validator, w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		tagErr(r.Context(), telemetry.ErrKindInvalidJSON)
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return false
	}
	if err := val.Validate(value); err != nil {
		tagErr(r.Context(), telemetry.ErrKindValidation)
		WriteJSON(w, http.StatusUnprocessableEntity, err)
		return false
	}
	return true
}

func tagErr(ctx context.Context, kind string) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", kind))
}
