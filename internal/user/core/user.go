// Package usercore contains the business logic for the user domain.
package usercore

import (
	"errors"

	usermodel "restful-boilerplate/internal/user/model"
)

// ErrNotFound is returned when a requested user does not exist.
var ErrNotFound = errors.New("user: not found")

// CreateUserInput is the input for creating a user.
type CreateUserInput = usermodel.CreateUserRequest

// UpdateUserInput is the input for updating a user.
type UpdateUserInput = usermodel.UpdateUserRequest
