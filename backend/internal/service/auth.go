package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
)

type AuthService struct {
	userRepo  repository.UserRepository
	authRepo  repository.AuthRepository
	jwtSecret []byte
}

func NewAuthService(userRepo repository.UserRepository, authRepo repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		authRepo:  authRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, error) {
	_, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return domain.User{}, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.UserRoleUser,
		Permissions:  []string{domain.PermissionMarketAccess},
	}

	return s.userRepo.Create(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken := domain.GenerateRandomString(32)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = s.authRepo.SaveRefreshToken(ctx, user.ID, refreshToken, expiresAt)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	userID, expiresAt, err := s.authRepo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	if time.Now().After(expiresAt) {
		_ = s.authRepo.DeleteRefreshToken(ctx, refreshToken)
		return "", "", ErrUnauthorized
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	newRefreshToken := domain.GenerateRandomString(32)
	newExpiresAt := time.Now().Add(7 * 24 * time.Hour)

	_ = s.authRepo.DeleteRefreshToken(ctx, refreshToken)
	err = s.authRepo.SaveRefreshToken(ctx, user.ID, newRefreshToken, newExpiresAt)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.authRepo.DeleteRefreshToken(ctx, refreshToken)
}

func (s *AuthService) generateAccessToken(user domain.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         user.ID,
		"email":       user.Email,
		"role":        user.Role,
		"permissions": user.Permissions,
		"exp":         time.Now().Add(15 * time.Minute).Unix(), // Shorter for access token
	})

	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *AuthService) ValidateToken(tokenString string) (int64, string, error) {
	claims, err := s.ValidateTokenClaims(tokenString)
	if err != nil {
		return 0, "", err
	}
	return claims.UserID, claims.Email, nil
}

type AuthClaims struct {
	UserID      int64
	Email       string
	Role        string
	Permissions []string
}

func (s *AuthService) ValidateTokenClaims(tokenString string) (AuthClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnauthorized
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return AuthClaims{}, ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return AuthClaims{}, ErrUnauthorized
	}

	userID, ok := parseInt64Claim(claims["sub"])
	if !ok {
		return AuthClaims{}, ErrUnauthorized
	}
	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return AuthClaims{}, ErrUnauthorized
	}
	role, _ := claims["role"].(string)
	if role == "" {
		role = domain.UserRoleUser
	}
	permissions := parsePermissionsClaim(claims["permissions"])

	return AuthClaims{UserID: userID, Email: email, Role: role, Permissions: permissions}, nil
}

func parseInt64Claim(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v != float64(int64(v)) {
			return 0, false
		}
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func (s *AuthService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.List(ctx)
}

func (s *AuthService) UpdateUserAccess(ctx context.Context, actorID int64, targetID int64, role string, permissions []string) (domain.User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return domain.User{}, err
	}
	if actor.Role != domain.UserRoleSuperAdmin && !hasPermission(actor.Permissions, domain.PermissionUserAdmin) {
		return domain.User{}, ErrForbidden
	}
	return s.userRepo.UpdateAccess(ctx, targetID, role, permissions)
}

func parsePermissionsClaim(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return []string{domain.PermissionMarketAccess}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{domain.PermissionMarketAccess}
	}
	return out
}

func hasPermission(items []string, permission string) bool {
	for _, item := range items {
		if item == permission {
			return true
		}
	}
	return false
}
