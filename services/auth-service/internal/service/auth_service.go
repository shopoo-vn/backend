package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace/auth-service/internal/model"
	"github.com/marketplace/auth-service/internal/repo"
	"github.com/marketplace/auth-service/internal/token"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidRefresh     = errors.New("invalid or expired refresh token")
	ErrUserInactive       = errors.New("user account is not active")
	ErrInvalidStatus      = errors.New("status must be 'active' or 'banned'")
)

type AuthService struct {
	users      *repo.UserRepo
	redis      *redis.Client
	tokens     *token.Manager
	refreshTTL time.Duration
}

func NewAuthService(users *repo.UserRepo, rdb *redis.Client, tm *token.Manager, refreshTTL time.Duration) *AuthService {
	return &AuthService{users: users, redis: rdb, tokens: tm, refreshTTL: refreshTTL}
}

// TokenPair is returned on login/refresh.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func refreshKey(t string) string { return "refresh:" + t }

func (s *AuthService) Register(ctx context.Context, email, phone, password, displayName string) (*model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Role:         "user",
		Status:       "active",
	}
	if phone = strings.TrimSpace(phone); phone != "" {
		u.Phone = &phone
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, *TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, nil, ErrInvalidCredentials
	}
	if u.Status != "active" {
		return nil, nil, ErrUserInactive
	}

	pair, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

func (s *AuthService) issueTokens(ctx context.Context, u *model.User) (*TokenPair, error) {
	access, exp, err := s.tokens.GenerateAccess(u.ID, u.Role)
	if err != nil {
		return nil, err
	}
	refresh, err := token.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	if err := s.redis.Set(ctx, refreshKey(refresh), u.ID, s.refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresAt: exp}, nil
}

// Refresh rotates the refresh token: the old one is invalidated and a new pair issued.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.redis.Get(ctx, refreshKey(refreshToken)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidRefresh
	}

	// Rotate: drop the presented token before minting a new one.
	if err := s.redis.Del(ctx, refreshKey(refreshToken)).Err(); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.redis.Del(ctx, refreshKey(refreshToken)).Err()
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID, displayName string, avatarURL *string) (*model.User, error) {
	return s.users.UpdateProfile(ctx, userID, displayName, avatarURL)
}

// UserPage is the paginated user list returned to the admin console.
type UserPage struct {
	Items []*model.User `json:"items"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}

func (s *AuthService) ListUsers(ctx context.Context, page, limit int) (*UserPage, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	items, err := s.users.List(ctx, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	total, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &UserPage{Items: items, Page: page, Limit: limit, Total: total}, nil
}

// SetUserStatus bans ("banned") or reinstates ("active") a user. Already-issued
// access tokens remain valid until they expire (~15m); Login blocks non-active users.
func (s *AuthService) SetUserStatus(ctx context.Context, id, status string) (*model.User, error) {
	if status != "active" && status != "banned" {
		return nil, ErrInvalidStatus
	}
	return s.users.UpdateStatus(ctx, id, status)
}
