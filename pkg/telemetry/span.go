package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanErr records err on the span, marks it as failed, and returns a wrapped error.
// Use in place of the span.RecordError + span.SetStatus + fmt.Errorf triple.
func SpanErr(span trace.Span, err error, op string) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return fmt.Errorf("%s: %w", op, err)
}
