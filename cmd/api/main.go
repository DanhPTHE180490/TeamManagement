package main

import (
	"log"
	"os"
	"path/filepath"

	"team-management/internal/auth"
	"team-management/internal/database"
	"team-management/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Connect to MySQL
	db := database.InitDB()
	defer db.Close()

	// 2. Build the Auth "Three-Layer Cake"
	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewAuthService(authRepo)
	authHandler := auth.NewAuthHandler(authService)

	// We will initialize the Team Cake here next...
	// teamRepo := team.NewTeamRepository(db)
	// teamService := team.NewTeamService(teamRepo)
	// teamHandler := team.NewTeamHandler(teamService)

	// 3. Setup Gin Router
	router := gin.Default()

	// Find the web directory (works whether running from project root or cmd/api)
	webDir := findWebDir()

	// Serve static files (frontend)
	router.Static("/static", filepath.Join(webDir, "static"))
	router.StaticFile("/", filepath.Join(webDir, "index.html"))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// ==========================================
	// 4. PUBLIC ROUTES
	// ==========================================
	authHandler.RegisterRoutes(router)

	// ==========================================
	// 5. PROTECTED ROUTES
	// ==========================================
	// We create a new group and attach the RequireAuth middleware to it.
	// Any route attached to 'protectedGroup' will require a valid JWT.
	protectedGroup := router.Group("/api")
	protectedGroup.Use(middleware.RequireAuth())
	{
		// This is just a test route to prove the Bouncer works and can read your ID!
		protectedGroup.GET("/me", func(c *gin.Context) {
			// Extracting the data the middleware securely placed in the context
			userID, _ := c.Get("userID")
			userRole, _ := c.Get("userRole")

			c.JSON(200, gin.H{
				"message": "You made it past the bouncer!",
				"user_id": userID,
				"role":    userRole,
			})
		})

		// teamHandler.RegisterRoutes(protectedGroup) // We will attach Teams here
	}

	// 6. Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func findWebDir() string {
	// Try current working directory first
	if _, err := os.Stat("web"); err == nil {
		return "web"
	}

	// Try going up one level (in case we're in cmd/api)
	if _, err := os.Stat("../web"); err == nil {
		return "../web"
	}

	if _, err := os.Stat("../../web"); err == nil {
		return "../../web"
	}

	log.Println("WARNING: Could not find web directory, defaulting to ./web")
	return "web"
}
