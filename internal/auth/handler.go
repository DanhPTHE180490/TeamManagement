package auth

import (
	"errors"
	"net/http"
	"strings"

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

	var requesterID int64
	if userIDCtx, ok := c.Get("userID"); ok {
		if floatID, isFloat := userIDCtx.(float64); isFloat {
			requesterID = int64(floatID)
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
