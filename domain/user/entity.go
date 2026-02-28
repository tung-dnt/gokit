// Package user defines the core domain types for user management.
package user

import "time"

// User is the core domain entity.
type User struct {
	ID        string    `json:"id"         example:"a1b2c3d4e5f6g7h8"`
	Name      string    `json:"name"       example:"Alice"`
	Email     string    `json:"email"      example:"alice@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// CreateUserInput carries the fields needed to create a new user.
type CreateUserInput struct {
	Name  string
	Email string
}

// UpdateUserInput carries the fields that may be changed on an existing user.
type UpdateUserInput struct {
	Name  string
	Email string
}
