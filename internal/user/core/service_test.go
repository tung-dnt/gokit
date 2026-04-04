package usercore_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	usercore "restful-boilerplate/internal/user/core"
	usermodel "restful-boilerplate/internal/user/model"
	pgdb "restful-boilerplate/pkg/postgres/db"
	"restful-boilerplate/pkg/testutil"
)

func newTestService(t *testing.T) *usercore.Service {
	t.Helper()
	pool := testutil.SetupPgTestDB(t)
	q := pgdb.New(pool)
	return usercore.NewService(q, noop.NewTracerProvider().Tracer("test"))
}

func TestCreateUser(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	u, err := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(u.ID) != 16 {
		t.Errorf("expected 16-char hex ID, got %q (len %d)", u.ID, len(u.ID))
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want %q", u.Name, "Alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "alice@example.com")
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "dup@example.com"})
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Bob", Email: "dup@example.com"})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestListUsers_Empty(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if users == nil {
		t.Fatal("expected non-nil slice for empty result")
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestListUsers_Multiple(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, _ = svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})
	_, _ = svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Bob", Email: "bob@example.com"})

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetUserByID_Found(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("ID = %q, want %q", u.ID, created.ID)
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want %q", u.Name, "Alice")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.GetUserByID(ctx, "nonexistent")
	if !errors.Is(err, usercore.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateUser_BothFields(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, usermodel.UpdateUserRequest{Name: "Bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if u.Name != "Bob" {
		t.Errorf("Name = %q, want %q", u.Name, "Bob")
	}
	if u.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "bob@example.com")
	}
}

func TestUpdateUser_NameOnly(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, usermodel.UpdateUserRequest{Name: "Bob"})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if u.Name != "Bob" {
		t.Errorf("Name = %q, want %q", u.Name, "Bob")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email should be preserved, got %q", u.Email)
	}
}

func TestUpdateUser_EmailOnly(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, usermodel.UpdateUserRequest{Email: "new@example.com"})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("Name should be preserved, got %q", u.Name)
	}
	if u.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "new@example.com")
	}
}

func TestUpdateUser_EmptyInput(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, usermodel.UpdateUserRequest{})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want %q", u.Name, "Alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "alice@example.com")
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.UpdateUser(ctx, "nonexistent", usermodel.UpdateUserRequest{Name: "Bob"})
	if !errors.Is(err, usercore.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, usermodel.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})

	if err := svc.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	_, err := svc.GetUserByID(ctx, created.ID)
	if !errors.Is(err, usercore.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.DeleteUser(ctx, "nonexistent")
	if !errors.Is(err, usercore.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
