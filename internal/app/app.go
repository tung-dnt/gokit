// Package app holds the shared application resources assembled once in main
// and injected into every domain module.
package app

import (
	"go.opentelemetry.io/otel/trace"

	sqlitedb "restful-boilerplate/pkg/sqlite/db"
)

// Validator validates struct fields. It is implemented by pkg/validator.CustomValidator.
type Validator interface {
	Validate(i any) error
}

// App is the single shared dependency container passed to every domain module.
// Modules use it to wire their own internal services.
// It intentionally holds a TracerProvider rather than a named Tracer so each
// module can create its own domain-specific tracer.
type App struct {
	Queries   *sqlitedb.Queries
	Validator Validator
	Tracer    trace.TracerProvider
}
