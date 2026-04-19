package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Error kinds used for the low-cardinality error.type span/metric attribute.
// Keep these stable — they're queried in SigNoz dashboards.
const (
	ErrKindNotFound     = "not_found"
	ErrKindValidation   = "validation"
	ErrKindInvalidJSON  = "invalid_json"
	ErrKindConflict     = "conflict"
	ErrKindUnauthorized = "unauthorized"
	ErrKindTimeout      = "timeout"
	ErrKindCanceled     = "canceled"
	ErrKindPanic        = "panic"
	ErrKindUnexpected   = "unexpected"
)

// SpanExpectedErr records a handled/expected failure on the span without marking
// it as errored. Per OTel HTTP semconv, server spans for 4xx outcomes (including
// 404 not found) MUST leave status Unset so error-rate SLIs aren't polluted.
// The caller supplies the low-cardinality kind (e.g. ErrKindNotFound).
func SpanExpectedErr(span trace.Span, err error, op, kind string) error {
	wrapped := fmt.Errorf("%s: %w", op, err)
	span.RecordError(wrapped)
	span.SetAttributes(attribute.String("error.type", kind))
	return wrapped
}

// SpanUnexpectedErr records an unexpected failure: marks the span Error, attaches
// a classified error.type attribute, and returns a wrapped error with op context.
// Use for any failure that the application cannot gracefully handle (DB errors,
// upstream 5xx, corrupt state).
func SpanUnexpectedErr(span trace.Span, err error, op string) error {
	wrapped := fmt.Errorf("%s: %w", op, err)
	span.RecordError(wrapped)
	span.SetAttributes(attribute.String("error.type", ClassifyErr(err)))
	span.SetStatus(codes.Error, wrapped.Error())
	return wrapped
}

// ClassifyErr returns a low-cardinality error kind for the given error.
// Extend as needed — labels flow into both span attributes and metric dimensions,
// so never use high-cardinality values like err.Error().
func ClassifyErr(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrKindCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrKindTimeout
	default:
		return ErrKindUnexpected
	}
}
