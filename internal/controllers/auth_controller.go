package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type AuthController struct{ authSvc *services.AuthService }

func NewAuthController(authSvc *services.AuthService) *AuthController {
	return &AuthController{authSvc: authSvc}
}

type registerOAuthRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
type loginOAuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (c *AuthController) RegisterOAuthUser(ctx *gin.Context) {
	var req registerOAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	user, pair, err := c.authSvc.RegisterOAuth(req.Username, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			merrors.Conflict(ctx, err.Error())
		} else {
			merrors.InternalServer(ctx, "registration failed")
		}
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"user": user, "token": pair.Token, "salt": pair.Salt})
}
func (c *AuthController) LoginOAuthUser(ctx *gin.Context) {
	var req loginOAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	user, pair, err := c.authSvc.LoginOAuth(req.Email, req.Password)
	if err != nil {
		merrors.Unauthorized(ctx, "invalid email or password")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"user": user, "token": pair.Token, "salt": pair.Salt})
}
func (c *AuthController) RegisterAnonymousUser(ctx *gin.Context) {
	id, pair, err := c.authSvc.RegisterAnonymous()
	if err != nil {
		merrors.InternalServer(ctx, "failed to create anonymous session")
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"anonymous_id": id, "token": pair.Token, "salt": pair.Salt})
}
func (c *AuthController) LogoutUser(ctx *gin.Context) {
	token, salt, ok := bearerCredentials(ctx)
	if !ok {
		merrors.Unauthorized(ctx, "a valid bearer token and X-Salt are required")
		return
	}
	if err := c.authSvc.RevokeToken(token, salt); err != nil {
		if strings.Contains(err.Error(), "invalid token") || strings.Contains(err.Error(), "session expired") {
			merrors.Unauthorized(ctx, "invalid or expired token")
			return
		}
		merrors.ServiceUnavailable(ctx, "token revocation is temporarily unavailable")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// ValidateTokenHandler handles POST /v1/auth/validate. Credentials are read only from headers.
func (c *AuthController) ValidateTokenHandler(ctx *gin.Context) {
	token, salt, ok := bearerCredentials(ctx)
	if !ok {
		merrors.Unauthorized(ctx, "a valid bearer token and X-Salt are required")
		return
	}
	subject, err := c.authSvc.ValidateToken(token, salt)
	if err != nil {
		if strings.Contains(err.Error(), "checking token status") || strings.Contains(err.Error(), "checking user session version") {
			merrors.ServiceUnavailable(ctx, "token validation is temporarily unavailable")
			return
		}
		merrors.Unauthorized(ctx, "invalid or expired token")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"subject": subject, "valid": true})
}
func bearerCredentials(ctx *gin.Context) (string, string, bool) {
	parts := strings.SplitN(ctx.GetHeader("Authorization"), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" || ctx.GetHeader("X-Salt") == "" {
		return "", "", false
	}
	return parts[1], ctx.GetHeader("X-Salt"), true
}
