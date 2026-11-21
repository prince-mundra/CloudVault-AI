// internal/services/auth_service.go
package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

const tokenExpiry = 24 * time.Hour

type AuthService struct {
	userRepo       *repository.UserRepository
	revocationRepo *repository.TokenRevocationRepository
	jwtSecret      string
}

func NewAuthService(userRepo *repository.UserRepository, revocationRepo *repository.TokenRevocationRepository, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, revocationRepo: revocationRepo, jwtSecret: jwtSecret}
}

type TokenPair struct {
	Token string `json:"token"`
	Salt  string `json:"salt"`
}

func (s *AuthService) RegisterOAuth(username, email, password string) (*models.User, *TokenPair, error) {
	if existing, err := s.userRepo.GetUserByEmail(email); err == nil && existing != nil {
		return nil, nil, errors.New("a user with that email already exists")
	}
	user, err := models.NewUser(generateID(), username, email, password)
	if err != nil {
		return nil, nil, fmt.Errorf("building user: %w", err)
	}
	if err := s.userRepo.CreateUser(user); err != nil {
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
	pair, err := s.issueToken(id, 0)
	if err != nil {
		return "", nil, err
	}
	return id, pair, nil
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
	if claims.SessionVersion == 0 && !strings.HasPrefix(claims.Subject, "anon_") {
		return "", errors.New("session expired")
	}
	if claims.SessionVersion > 0 {
		version, err := s.userRepo.GetAuthVersion(claims.Subject)
		if err != nil {
			return "", fmt.Errorf("checking user session version: %w", err)
		}
		if version != claims.SessionVersion {
			return "", errors.New("session expired")
		}
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
	if claims.SessionVersion == 0 && !strings.HasPrefix(claims.Subject, "anon_") {
		return errors.New("session expired")
	}
	if claims.SessionVersion > 0 {
		version, err := s.userRepo.GetAuthVersion(claims.Subject)
		if err != nil {
			return fmt.Errorf("checking user session version: %w", err)
		}
		if version != claims.SessionVersion {
			return errors.New("session expired")
		}
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
