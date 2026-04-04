// Package usermapping provides type conversions between DB models and HTTP response types
// for the user domain.
package usermapping

import (
	"time"

	pgdb "restful-boilerplate/pkg/postgres/db"
)

// UserResponse is the HTTP JSON response shape for a user.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a pgdb.User DB model to a UserResponse.
func ToResponse(u pgdb.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
