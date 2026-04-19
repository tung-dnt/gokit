// Package usermapping provides type conversions between DB models and HTTP response types
// for the user domain.
package user

import (
	"time"

	pgdb "gokit/pkg/postgres/db"
)

// userResponse is the HTTP JSON response shape for a user.
type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a pgdb.User DB model to a UserResponse.
func ToResponse(u pgdb.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
