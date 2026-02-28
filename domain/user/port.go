package user

import "context"

// Repository defines the persistence operations for users.
type Repository interface {
	Create(ctx context.Context, u *User) error
	List(ctx context.Context) ([]*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id string) error
}
