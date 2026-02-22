package user

import (
	"errors"
	"time"
)

var errNotFound = errors.New("user: not found")

// User is the core domain entity.
type User struct {
	ID        string    `json:"id"         example:"a1b2c3d4e5f6g7h8"`
	Name      string    `json:"name"       example:"Alice"`
	Email     string    `json:"email"      example:"alice@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

type createUserInput struct {
	Name  string
	Email string
}

type updateUserInput struct {
	Name  string
	Email string
}
