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

// CreateTeam godoc
// @Summary      Create a new team
// @Description  Allows a manager to create a new team
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      map[string]string  true  "Team Name (e.g., {'name': 'My Team'})"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Router       /api/teams/ [post]
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

// GetTeamByID godoc
// @Summary      Get team by ID
// @Description  Retrieve a team by its unique identifier
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/teams/{id} [get]
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

// GetTeamsByUserID godoc
// @Summary      Get teams by user ID
// @Description  Retrieve all teams associated with a specific user
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userID   path      int64  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/teams/users/{userID} [get]
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

// UpdateTeam godoc
// @Summary      Update a team
// @Description  Allows a manager to update a team's information
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Param        request  body      map[string]string  true  "Updated Team Name (e.g., {'name': 'Updated Team Name'})"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Router       /api/teams/{id} [put]
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

// DeleteTeam godoc
// @Summary      Delete a team
// @Description  Allows a manager to delete a team
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Router       /api/teams/{id} [delete]
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

// AddMemberToTeam godoc
// @Summary      Add member to team
// @Description  Allows a manager to add a user to a team
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Param        request  body      map[string]int64  true  "User ID (e.g., {'user_id': 123})"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Router       /api/teams/{id}/members [post]
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

// RemoveMemberFromTeam godoc
// @Summary      Remove member from team
// @Description  Allows a manager to remove a user from a team
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Param        userID   path      int64  true  "User ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
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

// UpdateMemberRole godoc
// @Summary      Update member role in team
// @Description  Allows a manager to update a team member's role (member, manager, main_manager)
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int64  true  "Team ID"
// @Param        userID   path      int64  true  "User ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
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
