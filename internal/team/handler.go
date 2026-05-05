package team

import (
	"net/http"
	"strconv"

	customErrors "team-management/internal/errors"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey         = "userID"
	ContextUserRoleKey       = "userRole"
	InvalidTeamIDError       = "Invalid team ID"
	InvalidUserIDError       = "Invalid user ID"
	UnauthorizedError        = "Unauthorized"
	ForbiddenError           = "Forbidden: insufficient permissions"
	TeamNotFoundError        = "Team not found"
	CreateTeamSuccessMessage = "Team created successfully"
	UpdateTeamSuccessMessage = "Team updated successfully"
	DeleteTeamSuccessMessage = "Team deleted successfully"
	DeleteTeamError          = "Failed to delete team"
	UpdateTeamError          = "Failed to update team"
	CreateTeamError          = "Failed to create team"
	GetTeamError             = "Failed to retrieve team"
	GetTeamsError            = "Failed to retrieve teams"
	InvalidUserIDTypeError   = "Invalid user ID type"
	InvalidInput             = "Invalid input"
)

type TeamHandler struct {
	service TeamService
}

func NewTeamHandler(service TeamService) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) RegisterRoutes(protectedGroup *gin.RouterGroup) {
	teamGroup := protectedGroup.Group("/teams")
	{
		teamGroup.POST("/", h.CreateTeam)
		teamGroup.GET("/:id", h.GetTeamByID)
		teamGroup.GET("/user/:userID", h.GetTeamsByUserID)
		teamGroup.PUT("/:id", h.UpdateTeam)
		teamGroup.DELETE("/:id", h.DeleteTeam)
		teamGroup.POST("/:id/members", h.AddMemberToTeam)
		teamGroup.DELETE("/:id/members/:userID", h.RemoveMemberFromTeam)
		teamGroup.PUT("/:id/members/:userID/role", h.UpdateMemberRole)
	}
}

func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidInput + ": " + err.Error()})
		return
	}

	userID, exists := c.Get(ContextUserIDKey)
	if !exists || userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	userRole, exists := c.Get(ContextUserRoleKey)
	if !exists || userRole == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	if userRole == "member" {
		c.JSON(http.StatusForbidden, gin.H{"error": ForbiddenError})
		return
	}

	floatID, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	uid := int64(floatID)
	role := userRole.(string)

	team, err := h.service.CreateTeam(req.Name, uid, role)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": CreateTeamError})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": CreateTeamSuccessMessage,
		"team":    team,
	})
}

func (h *TeamHandler) GetTeamByID(c *gin.Context) {
	idParam := c.Param("id")
	teamID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	team, err := h.service.GetTeamByID(teamID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": TeamNotFoundError})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetTeamError})
		return
	}

	if team == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": TeamNotFoundError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": team})
}

func (h *TeamHandler) GetTeamsByUserID(c *gin.Context) {
	userIDParam := c.Param("userID")
	userID, err := strconv.ParseInt(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidUserIDError})
		return
	}

	teams, err := h.service.GetTeamsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetTeamsError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	idParam := c.Param("id")
	teamID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidInput + ": " + err.Error()})
		return
	}

	userID, exists := c.Get(ContextUserIDKey)
	if !exists || userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	floatID, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}
	requesterID := int64(floatID)

	team, err := h.service.UpdateTeam(teamID, req.Name, requesterID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": TeamNotFoundError})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": UpdateTeamError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": UpdateTeamSuccessMessage,
		"team":    team,
	})
}

func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	idParam := c.Param("id")
	teamID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	userID, exists := c.Get(ContextUserIDKey)
	if !exists || userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	floatID, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}
	requesterID := int64(floatID)

	err = h.service.DeleteTeam(teamID, requesterID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": TeamNotFoundError})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": DeleteTeamError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": DeleteTeamSuccessMessage})
}

func (h *TeamHandler) AddMemberToTeam(c *gin.Context) {
	teamIDParam := c.Param("id")
	teamID, err := strconv.ParseInt(teamIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidInput + ": " + err.Error()})
		return
	}

	userID, exists := c.Get(ContextUserIDKey)
	if !exists || userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	floatID, ok := userID.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}
	requesterID := int64(floatID)

	err = h.service.AddMemberToTeam(teamID, req.UserID, requesterID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member added to team"})
}

func (h *TeamHandler) RemoveMemberFromTeam(c *gin.Context) {
	teamIDParam := c.Param("id")
	teamID, err := strconv.ParseInt(teamIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	targetUserIDParam := c.Param("userID")
	targetUserID, err := strconv.ParseInt(targetUserIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidUserIDError})
		return
	}

	requesterCtx, exists := c.Get(ContextUserIDKey)
	if !exists || requesterCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	floatID, ok := requesterCtx.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}
	requesterID := int64(floatID)

	err = h.service.RemoveMemberFromTeam(teamID, targetUserID, requesterID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member removed from team"})
}

func (h *TeamHandler) UpdateMemberRole(c *gin.Context) {
	teamIDParam := c.Param("id")
	teamID, err := strconv.ParseInt(teamIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidTeamIDError})
		return
	}

	targetUserIDParam := c.Param("userID")
	targetUserID, err := strconv.ParseInt(targetUserIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidUserIDError})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=member manager main_manager"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": InvalidInput + ": " + err.Error()})
		return
	}

	requesterCtx, exists := c.Get(ContextUserIDKey)
	if !exists || requesterCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	floatID, ok := requesterCtx.(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}
	requesterID := int64(floatID)

	err = h.service.UpdateMemberRole(teamID, targetUserID, req.Role, requesterID)
	if err != nil {
		if customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member role updated"})
}
