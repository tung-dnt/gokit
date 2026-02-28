package user

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"restful-boilerplate/domain/user"
	"restful-boilerplate/infra/sqlite/userrepo"
	testutil "restful-boilerplate/infra/testutil"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := userrepo.NewSQLite(db)
	return NewService(repo, noop.NewTracerProvider().Tracer("test"))
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	u, err := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "dup@example.com"})
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = svc.CreateUser(ctx, user.CreateUserInput{Name: "Bob", Email: "dup@example.com"})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestListUsers_Empty(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	_, _ = svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})
	_, _ = svc.CreateUser(ctx, user.CreateUserInput{Name: "Bob", Email: "bob@example.com"})

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetUserByID_Found(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.GetUserByID(ctx, "nonexistent")
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateUser_BothFields(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, user.UpdateUserInput{Name: "Bob", Email: "bob@example.com"})
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, user.UpdateUserInput{Name: "Bob"})
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, user.UpdateUserInput{Email: "new@example.com"})
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.UpdateUser(ctx, created.ID, user.UpdateUserInput{})
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
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.UpdateUser(ctx, "nonexistent", user.UpdateUserInput{Name: "Bob"})
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})

	if err := svc.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	// Verify user is gone.
	_, err := svc.GetUserByID(ctx, created.ID)
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.DeleteUser(ctx, "nonexistent")
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	t.Parallel()

	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if len(id) != 16 {
		t.Errorf("expected 16-char string, got %q (len %d)", id, len(id))
	}
	if _, decErr := hex.DecodeString(id); decErr != nil {
		t.Errorf("expected valid hex string, got %q: %v", id, decErr)
	}
}

func TestGenerateID_Unique(t *testing.T) {
	t.Parallel()

	id1, _ := generateID()
	id2, _ := generateID()
	if id1 == id2 {
		t.Errorf("expected unique IDs, got %q twice", id1)
	}
}
