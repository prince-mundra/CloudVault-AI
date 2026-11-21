// internal/server/routes.go
package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/controllers"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/services"
	"github.com/souvik03-136/Go-Store/internal/storage"
)

// initRoutes wires the existing API plus Phase 1 controls. The returned worker
// is owned by Server.Start and exits when its context is cancelled.
func initRoutes(router *gin.Engine, db *sql.DB, store storage.Storage, jwtSecret string, quotaBytes, maxUploadBytes int64, allowedOrigins []string, aiSearcher controllers.SemanticSearcher) func(context.Context) {
	userRepo := repository.NewUserRepository(db)
	fileRepo := repository.NewFileRepository(db)
	folderRepo := repository.NewFolderRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	revocationRepo := repository.NewTokenRevocationRepository(db)
	activityRepo := repository.NewFileActivityRepository(db)
	uploadRepo := repository.NewUploadRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	deleteRepo := repository.NewFileDeleteRepository(db)
	shareLinkRepo := repository.NewShareLinkRepository(db)
	ingestionRepo := repository.NewIngestionRepository(db)
	summaryRepo := repository.NewSummaryRepository(db)
	authSvc := services.NewAuthService(userRepo, revocationRepo, jwtSecret)
	userSvc := services.NewUserService(userRepo)
	folderSvc := services.NewFolderService(folderRepo)
	fileSvc := services.NewFileService(fileRepo, folderRepo, permissionRepo, activityRepo, uploadRepo, auditRepo, deleteRepo, store, quotaBytes, maxUploadBytes)
	sharingSvc := services.NewSharingService(fileRepo, userRepo, permissionRepo, auditRepo)
	analyticsSvc := services.NewAnalyticsService(activityRepo, quotaBytes)
	shareLinkSvc := services.NewShareLinkService(fileRepo, shareLinkRepo, store, auditRepo, activityRepo)
	authCtrl := controllers.NewAuthController(authSvc)
	userCtrl := controllers.NewUserController(userSvc)
	folderCtrl := controllers.NewFolderController(folderSvc)
	fileCtrl := controllers.NewFileController(fileSvc, maxUploadBytes)
	sharingCtrl := controllers.NewSharingController(sharingSvc)
	analyticsCtrl := controllers.NewAnalyticsController(analyticsSvc)
	shareLinkCtrl := controllers.NewShareLinkController(shareLinkSvc)
	semanticSearchCtrl := controllers.NewSemanticSearchController(aiSearcher)
	ingestionCtrl := controllers.NewIngestionController(ingestionRepo)
	summaryCtrl := controllers.NewSummaryController(summaryRepo)

	router.Use(auth.CORSMiddleware(allowedOrigins))
	router.Use(auth.RequestLogger())
	router.GET("/healthz", func(ctx *gin.Context) {
		if db == nil || db.PingContext(ctx.Request.Context()) != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/v1")
	authGroup := v1.Group("/auth")
	registerLimit := auth.NewIPRateLimiter(5, time.Hour)
	loginLimit := auth.NewIPRateLimiter(10, time.Minute)
	anonymousLimit := auth.NewIPRateLimiter(20, time.Minute)
	validateLimit := auth.NewIPRateLimiter(30, time.Minute)
	authGroup.POST("/oauth/register", registerLimit.Middleware(), authCtrl.RegisterOAuthUser)
	authGroup.POST("/oauth/login", loginLimit.Middleware(), authCtrl.LoginOAuthUser)
	authGroup.POST("/anonymous/register", anonymousLimit.Middleware(), authCtrl.RegisterAnonymousUser)
	authGroup.POST("/validate", validateLimit.Middleware(), authCtrl.ValidateTokenHandler)
	publicLinks := v1.Group("/public/share-links")
	publicLinkLimit := auth.NewIPRateLimiter(20, time.Minute)
	publicLinks.GET("/:token/download", publicLinkLimit.Middleware(), shareLinkCtrl.Download)

	protected := v1.Group("")
	protected.Use(auth.JWTAuthMiddleware(jwtSecret, revocationRepo, userRepo))
	protected.POST("/auth/logout", authCtrl.LogoutUser)
	protected.GET("/analytics/storage", analyticsCtrl.GetStorage)
	semanticSearchLimit := auth.NewIPRateLimiter(20, time.Minute)
	protected.POST("/search/semantic", semanticSearchLimit.Middleware(), semanticSearchCtrl.Search)
	admin := protected.Group("/admin")
	admin.Use(auth.AdminRoleMiddleware(userRepo))
	admin.GET("/users", userCtrl.ListUsers)
	users := protected.Group("/users")
	users.GET("/:id", userCtrl.GetUserByID)
	users.PUT("/:id", userCtrl.UpdateUser)
	users.DELETE("/:id", userCtrl.DeleteUser)
	folders := protected.Group("/folders")
	folders.POST("", folderCtrl.Create)
	folders.GET("", folderCtrl.List)
	folders.PATCH("/:id", folderCtrl.Update)
	folders.DELETE("/:id", folderCtrl.Delete)
	protected.GET("/shares/with-me", sharingCtrl.SharedWithMe)
	protected.DELETE("/shares/with-me/:id", sharingCtrl.Leave)
	files := protected.Group("/files")
	files.POST("", fileCtrl.CreateFile)
	files.GET("", fileCtrl.ListMyFiles)
	files.GET("/:id/download", fileCtrl.DownloadFile)
	files.GET("/:id/ingestion", ingestionCtrl.GetStatus)
	files.GET("/:id/summary", summaryCtrl.GetStatus)
	files.PATCH("/:id/folder", fileCtrl.MoveToFolder)
	files.GET("/:id", fileCtrl.GetFileByID)
	files.PUT("/:id", fileCtrl.UpdateFile)
	files.DELETE("/:id", fileCtrl.DeleteFile)
	files.POST("/:id/shares", sharingCtrl.Grant)
	files.GET("/:id/shares", sharingCtrl.List)
	files.DELETE("/:id/shares/:user_id", sharingCtrl.Revoke)
	files.POST("/:id/share-links", shareLinkCtrl.Create)
	files.GET("/:id/share-links", shareLinkCtrl.List)
	files.DELETE("/:id/share-links/:link_id", shareLinkCtrl.Revoke)

	return func(ctx context.Context) {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := fileSvc.RunMaintenance(ctx); err != nil && ctx.Err() == nil {
					log.Printf("file maintenance pass failed: %v", err)
				}
			}
		}
	}
}
