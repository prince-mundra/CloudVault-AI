// internal/auth/middleware.go
package auth

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ClaimsKey  = "claims"
	SubjectKey = "subject"
)

type TokenRevocationChecker interface {
	IsRevoked(tokenHash string) (bool, error)
}
type UserRoleChecker interface {
	GetUserRole(userID string) (string, error)
}
type UserSessionVersionChecker interface {
	GetAuthVersion(userID string) (int64, error)
}

// CORSMiddleware only reflects an explicitly configured origin.
func CORSMiddleware(allowedOrigins ...[]string) gin.HandlerFunc {
	allowed := map[string]struct{}{"http://localhost:5173": {}, "http://127.0.0.1:5173": {}}
	if len(allowedOrigins) > 0 {
		allowed = make(map[string]struct{}, len(allowedOrigins[0]))
		for _, origin := range allowedOrigins[0] {
			allowed[origin] = struct{}{}
		}
	}
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		ctx.Header("Vary", "Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Salt")
			ctx.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Content-Disposition")
		}
		if ctx.Request.Method == http.MethodOptions {
			if origin != "" {
				ctx.AbortWithStatus(http.StatusNoContent)
				return
			}
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		route := ctx.FullPath()
		if route == "" {
			route = "unmatched"
		}
		log.Printf("http method=%s route=%s status=%d latency=%s", ctx.Request.Method, route, ctx.Writer.Status(), time.Since(start).Round(time.Millisecond))
	}
}

func JWTAuthMiddleware(jwtSecret string, revocations TokenRevocationChecker, versions ...UserSessionVersionChecker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.SplitN(ctx.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "a bearer token is required"})
			return
		}
		tokenString := parts[1]
		salt := ctx.GetHeader("X-Salt")
		if salt == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Salt header is required"})
			return
		}
		claims, err := ValidateToken(jwtSecret, tokenString, salt)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if revocations != nil {
			revoked, err := revocations.IsRevoked(TokenFingerprint(tokenString))
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service unavailable"})
				return
			}
			if revoked {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
				return
			}
		}
		if claims.SessionVersion == 0 && !strings.HasPrefix(claims.Subject, "anon_") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}
		if claims.SessionVersion > 0 && len(versions) > 0 && versions[0] != nil {
			current, err := versions[0].GetAuthVersion(claims.Subject)
			if err != nil {
				if err.Error() == "user not found" {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account no longer exists"})
					return
				}
				ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service unavailable"})
				return
			}
			if current != claims.SessionVersion {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
				return
			}
		}
		ctx.Set(ClaimsKey, claims)
		ctx.Set(SubjectKey, claims.Subject)
		ctx.Next()
	}
}

func AdminRoleMiddleware(roles UserRoleChecker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := ctx.Get(SubjectKey)
		id, valid := userID.(string)
		if !ok || !valid || id == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		role, err := roles.GetUserRole(id)
		if err != nil {
			if err.Error() == "user not found" {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator role required"})
				return
			}
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authorization service unavailable"})
			return
		}
		if role != "admin" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator role required"})
			return
		}
		ctx.Set("role", role)
		ctx.Next()
	}
}
