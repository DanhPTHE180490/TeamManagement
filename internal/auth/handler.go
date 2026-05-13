package auth

import (
	"net/http"
	"strings"

	customErrors "team-management/internal/errors"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP requests for authentication
type AuthHandler struct {
	service AuthService
}

var maxBulkImportUploadBytes int64 = 10 * 1024 * 1024

// NewAuthHandler is the constructor
func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// RegisterRoutes connects the Gin router to these functions
func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
	}
}

func (h *AuthHandler) RegisterProtectedRoutes(protectedGroup *gin.RouterGroup) {
	authGroup := protectedGroup.Group("/auth")
	{
		authGroup.POST("/import-users", h.BulkImportUsers)
	}
}

// Register expects {"username": "...", "email": "...", "password": "...", "role": "manager"}
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,max=50"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role" binding:"required,oneof=manager member admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	user, err := h.service.Register(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login expects {"email": "...", "password": "..."}
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Call the Service Layer to verify and get the JWT
	token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) || customErrors.IsErrorType(err, customErrors.ErrTypeUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Using JWT is stateless so we can't invalidate tokens server-side without additional infrastructure (like a blacklist).
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful (client should delete the token)",
	})
}

func (h *AuthHandler) BulkImportUsers(c *gin.Context) {
	userRole, exists := c.Get("userRole")
	if !exists || (userRole != "manager" && userRole != "main_manager") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: only managers can bulk import users"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBulkImportUploadBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large: maximum upload size exceeded"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get file from request: " + err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file stream"})
		return
	}
	defer file.Close()

	summary, err := h.service.BulkImportUsersFromCSV(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "processing failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Import complete",
		"summary": summary,
	})
}
