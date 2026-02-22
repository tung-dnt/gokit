package dto

type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=1,max=100"  example:"Alice"`
	Email string `json:"email" validate:"required,email"           example:"alice@example.com"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"  validate:"omitempty,min=1,max=100"  example:"Alice"`
	Email string `json:"email" validate:"omitempty,email"           example:"alice@example.com"`
}
