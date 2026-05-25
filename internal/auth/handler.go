package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"team-management/internal/models"
	"team-management/internal/utils"

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
		authGroup.POST("/logout", h.Logout)
		authGroup.POST("/import-users", h.BulkImportUsers)
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account with a specified role (manager, member)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Example: {'username': 'test', 'email': 't@t.com', 'password': '123', 'role': 'manager'}"
// @Success      201      {object}  map[string]interface{}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.UserRegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	ctx := c.Request.Context()

	user, err := h.service.Register(ctx, req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			if appErr.Type == utils.ErrTypeInternal {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": utils.ErrTypeInternalServer})
			} else {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login godoc
// @Summary      Login a user
// @Description  Authenticates a user and returns a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Example: {'email': 't@t.com', 'password': '123'}"
// @Success      200      {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.UserLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	// Call the Service Layer to verify and get the JWT
	token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			if appErr.Type == utils.ErrTypeInternal {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": utils.ErrTypeInternalServer})
			} else {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

// internal/auth/handler.go

// Logout godoc
// @Summary      Logout a user
// @Description  Invalidates the user's current bearer token server-side.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Router       /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Extract the raw token from the header
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Assuming your middleware parsed the JWT and stored the expiration in the context
	// (If it doesn't do this yet, you'll need to add it to your middleware!)
	expCtx, exists := c.Get("tokenExp")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not determine token expiration"})
		return
	}

	expiration, ok := expCtx.(time.Time)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid expiration format"})
		return
	}

	// Send it to Redis!
	err := h.service.BlacklistToken(c.Request.Context(), tokenString, expiration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout securely"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful! Token invalidated."})
}

// BulkImportUsers godoc
// @Summary      Bulk import users
// @Description  Allows managers or main managers to bulk import users via a CSV file upload.
// @Tags         Auth
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file     formData  file  true  "CSV file containing user data"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      413      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /api/auth/import-users [post]
func (h *AuthHandler) BulkImportUsers(c *gin.Context) {
	userRole, exists := c.Get("userRole")
	if !exists || (userRole != "manager" && userRole != "main_manager") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: only managers can bulk import users"})
		return
	}

	_, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing user ID"})
		return
	}

	var requesterID int64
	if userIDCtx, ok := c.Get("userID"); ok {
		if floatID, isFloat := userIDCtx.(float64); isFloat {
			requesterID = int64(floatID)
		}
		switch v := userIDCtx.(type) {
		case int64:
			requesterID = v
		case int:
			requesterID = int64(v)
		case int32:
			requesterID = int64(v)
		case float64:
			requesterID = int64(v)
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid user ID"})
			return
		}
		if requesterID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid user ID"})
			return
		}
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBulkImportUploadBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large: maximum upload size exceeded"})
			return
		}
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
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

	summary, err := h.service.BulkImportUsersFromCSV(c.Request.Context(), requesterID, file)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Import complete",
		"summary": summary,
	})
}
