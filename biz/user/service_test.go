package user

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	testutil "restful-boilerplate/dx/test"
	sqlitedb "restful-boilerplate/repo/sqlite/db"
)

func newTestService(t *testing.T) *userService {
	t.Helper()
	db := testutil.SetupTestDB(t)
	return &userService{
		q:      sqlitedb.New(db),
		tracer: noop.NewTracerProvider().Tracer("test"),
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	u, err := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("createUser() error = %v", err)
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

	_, err := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "dup@example.com"})
	if err != nil {
		t.Fatalf("first createUser() error = %v", err)
	}

	_, err = svc.createUser(ctx, createUserInput{Name: "Bob", Email: "dup@example.com"})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestListUsers_Empty(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	users, err := svc.listUsers(ctx)
	if err != nil {
		t.Fatalf("listUsers() error = %v", err)
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

	_, _ = svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})
	_, _ = svc.createUser(ctx, createUserInput{Name: "Bob", Email: "bob@example.com"})

	users, err := svc.listUsers(ctx)
	if err != nil {
		t.Fatalf("listUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetUserByID_Found(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.getUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("getUserByID() error = %v", err)
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

	_, err := svc.getUserByID(ctx, "nonexistent")
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
	}
}

func TestUpdateUser_BothFields(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.updateUser(ctx, created.ID, updateUserInput{Name: "Bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("updateUser() error = %v", err)
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

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.updateUser(ctx, created.ID, updateUserInput{Name: "Bob"})
	if err != nil {
		t.Fatalf("updateUser() error = %v", err)
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

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.updateUser(ctx, created.ID, updateUserInput{Email: "new@example.com"})
	if err != nil {
		t.Fatalf("updateUser() error = %v", err)
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

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	u, err := svc.updateUser(ctx, created.ID, updateUserInput{})
	if err != nil {
		t.Fatalf("updateUser() error = %v", err)
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

	_, err := svc.updateUser(ctx, "nonexistent", updateUserInput{Name: "Bob"})
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	created, _ := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

	if err := svc.deleteUser(ctx, created.ID); err != nil {
		t.Fatalf("deleteUser() error = %v", err)
	}

	// Verify user is gone.
	_, err := svc.getUserByID(ctx, created.ID)
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound after delete, got %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.deleteUser(ctx, "nonexistent")
	if !errors.Is(err, errNotFound) {
		t.Errorf("expected errNotFound, got %v", err)
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

func TestToUser(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	row := sqlitedb.User{ID: "abc123", Name: "Alice", Email: "alice@example.com", CreatedAt: now}
	u := toUser(row)

	if u.ID != "abc123" {
		t.Errorf("ID = %q, want %q", u.ID, "abc123")
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want %q", u.Name, "Alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "alice@example.com")
	}
	if !u.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", u.CreatedAt, now)
	}
}
