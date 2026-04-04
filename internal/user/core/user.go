// Package usercore contains the business logic for the user domain.
package usercore

import (
	"errors"
)

// ErrNotFound is returned when a requested user does not exist.
var ErrNotFound = errors.New("user: not found")
