package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"
	"invest/backend/internal/repository/postgres"
)

func TestAuthService_Register(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	authRepo := postgres.NewAuthRepository(db)
	svc := NewAuthService(userRepo, authRepo, "secret")

	email := "test@example.com"
	password := "password123"

	mock.ExpectQuery("SELECT id, email, password_hash, COALESCE\\(role, 'user'\\), COALESCE\\(permissions, '\\[\"market_access\"\\]'::jsonb\\), created_at, updated_at").
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs(email, sqlmock.AnyArg(), "user", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "permissions"}).AddRow(1, "user", []byte(`["market_access"]`)))

	user, err := svc.Register(context.Background(), email, password)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}
}

func TestAuthService_Login(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	authRepo := postgres.NewAuthRepository(db)
	svc := NewAuthService(userRepo, authRepo, "secret")

	email := "test@example.com"
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	mock.ExpectQuery("SELECT id, email, password_hash, COALESCE\\(role, 'user'\\), COALESCE\\(permissions, '\\[\"market_access\"\\]'::jsonb\\), created_at, updated_at").
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "role", "permissions", "created_at", "updated_at"}).
			AddRow(1, email, string(hash), "super_admin", []byte(`["market_access","ml_admin","user_admin"]`), time.Now(), time.Now()))

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	accessToken, refreshToken, err := svc.Login(context.Background(), email, password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if accessToken == "" || refreshToken == "" {
		t.Error("expected tokens, got empty")
	}

	userID, userEmail, err := svc.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if userID != 1 || userEmail != email {
		t.Errorf("token validation failed: %d, %s", userID, userEmail)
	}
}
