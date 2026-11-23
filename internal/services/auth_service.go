// internal/services/auth_service.go
package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/models"
)

const tokenExpiry = 24 * time.Hour

var (
	ErrEmailAlreadyExists    = errors.New("a user with that email already exists")
	ErrUsernameAlreadyExists = errors.New("a user with that username already exists")
)

type UserAccountStore interface {
	CreateUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetAuthVersion(id string) (int64, error)
	GetAuthInfo(id string) (int64, bool, error)
}

type TokenRevocationStore interface {
	IsRevoked(tokenHash string) (bool, error)
	Revoke(tokenHash string, expiresAt time.Time) error
}

type AuthService struct {
	userRepo       UserAccountStore
	revocationRepo TokenRevocationStore
	jwtSecret      string
}

func NewAuthService(userRepo UserAccountStore, revocationRepo TokenRevocationStore, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, revocationRepo: revocationRepo, jwtSecret: jwtSecret}
}

type TokenPair struct {
	Token string `json:"token"`
	Salt  string `json:"salt"`
}

func (s *AuthService) RegisterOAuth(username, email, password string) (*models.User, *TokenPair, error) {
	if existing, err := s.userRepo.GetUserByEmail(email); err == nil && existing != nil {
		return nil, nil, ErrEmailAlreadyExists
	}
	if existing, err := s.userRepo.GetUserByUsername(username); err == nil && existing != nil {
		return nil, nil, ErrUsernameAlreadyExists
	}
	user, err := models.NewUser(generateID(), username, email, password)
	if err != nil {
		return nil, nil, fmt.Errorf("building user: %w", err)
	}
	if err := s.userRepo.CreateUser(user); err != nil {
		if strings.Contains(err.Error(), "users_username_key") {
			return nil, nil, ErrUsernameAlreadyExists
		}
		if strings.Contains(err.Error(), "users_email_key") {
			return nil, nil, ErrEmailAlreadyExists
		}
		return nil, nil, fmt.Errorf("persisting user: %w", err)
	}
	pair, err := s.issueToken(user.ID, user.AuthVersion)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}
func (s *AuthService) LoginOAuth(email, password string) (*models.User, *TokenPair, error) {
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}
	if user.Role == "anonymous" || user.IsAnonymous {
		return nil, nil, errors.New("invalid credentials")
	}
	if !user.CheckPassword(password) {
		return nil, nil, errors.New("invalid credentials")
	}
	pair, err := s.issueToken(user.ID, user.AuthVersion)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}
func (s *AuthService) RegisterAnonymous() (string, *TokenPair, error) {
	id := auth.GenerateAnonymousID()
	user, err := models.NewAnonymousUser(id)
	if err != nil {
		return "", nil, fmt.Errorf("building anonymous user: %w", err)
	}
	if err := s.userRepo.CreateUser(user); err != nil {
		return "", nil, fmt.Errorf("persisting anonymous user: %w", err)
	}
	token, salt, err := auth.GenerateTokenWithVersionAndAnonymous(s.jwtSecret, user.ID, user.AuthVersion, true, tokenExpiry)
	if err != nil {
		return "", nil, fmt.Errorf("issuing anonymous token: %w", err)
	}
	return user.ID, &TokenPair{Token: token, Salt: salt}, nil
}

func (s *AuthService) ValidateToken(tokenString, salt string) (string, error) {
	claims, err := auth.ValidateToken(s.jwtSecret, tokenString, salt)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	revoked, err := s.revocationRepo.IsRevoked(auth.TokenFingerprint(tokenString))
	if err != nil {
		return "", fmt.Errorf("checking token status: %w", err)
	}
	if revoked {
		return "", errors.New("token revoked")
	}
	if claims.SessionVersion <= 0 {
		return "", errors.New("session expired")
	}
	version, isAnon, err := s.userRepo.GetAuthInfo(claims.Subject)
	if err != nil {
		return "", fmt.Errorf("checking user session version: %w", err)
	}
	if version != claims.SessionVersion {
		return "", errors.New("session expired")
	}
	if claims.IsAnonymous != isAnon {
		return "", errors.New("token identity mismatch")
	}
	return claims.Subject, nil
}
func (s *AuthService) RevokeToken(tokenString, salt string) error {
	claims, err := auth.ValidateToken(s.jwtSecret, tokenString, salt)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	if claims.ExpiresAt == nil {
		return errors.New("token expiration is required")
	}
	if claims.SessionVersion <= 0 {
		return errors.New("session expired")
	}
	version, isAnon, err := s.userRepo.GetAuthInfo(claims.Subject)
	if err != nil {
		return fmt.Errorf("checking user session version: %w", err)
	}
	if version != claims.SessionVersion {
		return errors.New("session expired")
	}
	if claims.IsAnonymous != isAnon {
		return errors.New("token identity mismatch")
	}
	return s.revocationRepo.Revoke(auth.TokenFingerprint(tokenString), claims.ExpiresAt.Time)
}
func (s *AuthService) issueToken(subject string, version int64) (*TokenPair, error) {
	token, salt, err := auth.GenerateTokenWithVersion(s.jwtSecret, subject, version, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}
	return &TokenPair{Token: token, Salt: salt}, nil
}
