package team

import (
	"errors"
	"net/http"
	"strconv"

	"team-management/internal/utils"

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
	InvalidUserRoleTypeError = "Invalid user role type"
	InvalidInput             = "Invalid input"
)

type TeamHandler struct {
	service TeamService
}

func getRequesterID(c *gin.Context) (int64, bool, bool) {
	requesterCtx, exists := c.Get(ContextUserIDKey)
	if !exists || requesterCtx == nil {
		return 0, false, false
	}
	userIDFloat, ok := requesterCtx.(float64)
	if !ok {
		return 0, true, false
	}
	return int64(userIDFloat), true, true
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
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	userID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	userRole, exists := c.Get(ContextUserRoleKey)
	if !exists || userRole == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}

	role, ok := userRole.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserRoleTypeError})
		return
	}

	if role == "member" {
		c.JSON(http.StatusForbidden, gin.H{"error": ForbiddenError})
		return
	}

	team, err := h.service.CreateTeam(c.Request.Context(), req.Name, userID, role)
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

	team, err := h.service.GetTeamByID(c.Request.Context(), teamID)
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

	teams, err := h.service.GetTeamsByUserID(c.Request.Context(), userID)
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
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	requesterID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	team, err := h.service.UpdateTeam(c.Request.Context(), teamID, req.Name, requesterID)
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

	requesterID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	err = h.service.DeleteTeam(c.Request.Context(), teamID, requesterID)
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
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	requesterID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	err = h.service.AddMemberToTeam(c.Request.Context(), teamID, req.UserID, requesterID)
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

	requesterID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	err = h.service.RemoveMemberFromTeam(c.Request.Context(), teamID, targetUserID, requesterID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
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
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   appErr.Message,
			"details": appErr.Details,
		})
		return
	}

	requesterID, exists, ok := getRequesterID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": UnauthorizedError})
		return
	}
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": InvalidUserIDTypeError})
		return
	}

	err = h.service.UpdateMemberRole(c.Request.Context(), teamID, targetUserID, req.Role, requesterID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member role updated"})
}
