package dto

// CreateUserRequest holds the input fields for creating a new user.
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=1,max=100"  example:"Alice"`
	Email string `json:"email" validate:"required,email"           example:"alice@example.com"`
}

// UpdateUserRequest holds the input fields for updating an existing user.
type UpdateUserRequest struct {
	Name  string `json:"name"  validate:"omitempty,min=1,max=100"  example:"Alice"`
	Email string `json:"email" validate:"omitempty,email"           example:"alice@example.com"`
}
