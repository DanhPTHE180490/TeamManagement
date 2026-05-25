package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"team-management/internal/asset"
	"team-management/internal/auth"
	"team-management/internal/database"
	"team-management/internal/middleware"
	"team-management/internal/team"

	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.InitDB()
	defer db.Close()

	// Initialize Redis client for caching
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewAuthService(authRepo)
	authHandler := auth.NewAuthHandler(authService)

	teamRepo := team.NewTeamRepository(db, redisClient)
	teamService := team.NewTeamService(teamRepo)
	teamHandler := team.NewTeamHandler(teamService)

	assetRepo := asset.NewAssetRepository(db, redisClient)
	assetService := asset.NewAssetService(assetRepo)
	assetHandler := asset.NewAssetHandler(assetService)

	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	webDir := findWebDir()

	router.Static("/static", filepath.Join(webDir, "static"))
	router.StaticFile("/", filepath.Join(webDir, "index.html"))
	router.StaticFile("/team.html", filepath.Join(webDir, "team.html"))
	router.StaticFile("/notes.html", filepath.Join(webDir, "notes.html"))
	router.StaticFile("/import-users.html", filepath.Join(webDir, "import-users.html"))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	authHandler.RegisterRoutes(router)

	protectedGroup := router.Group("/api")
	protectedGroup.Use(middleware.RequireAuth())
	{
		protectedGroup.GET("/me", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			userRole, _ := c.Get("userRole")

			c.JSON(200, gin.H{
				"message": "You made it past the bouncer!",
				"user_id": userID,
				"role":    userRole,
			})
		})
		authHandler.RegisterProtectedRoutes(protectedGroup)
		teamHandler.RegisterRoutes(protectedGroup)
		assetHandler.RegisterRoutes(protectedGroup)
	}

	log.Println("Server starting on http://localhost:8080")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}
}

func findWebDir() string {
	if _, err := os.Stat("web"); err == nil {
		return "web"
	}

	if _, err := os.Stat("../web"); err == nil {
		return "../web"
	}

	if _, err := os.Stat("../../web"); err == nil {
		return "../../web"
	}

	log.Println("WARNING: Could not find web directory, defaulting to ./web")
	return "web"
}
