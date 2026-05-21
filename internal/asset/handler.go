package asset

import (
	"errors"
	"net/http"
	"strconv"

	"team-management/internal/models"
	"team-management/internal/utils"

	"github.com/gin-gonic/gin"
)

type AssetHandler struct {
	service AssetService
}

func NewAssetHandler(service AssetService) *AssetHandler {
	return &AssetHandler{service: service}
}

func getRequesterID(c *gin.Context) (int64, bool) {
	requesterCtx, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	userIDFloat, ok := requesterCtx.(float64)
	if !ok {
		return 0, false
	}
	return int64(userIDFloat), true
}

func (h *AssetHandler) RegisterRoutes(protectedGroup *gin.RouterGroup) {
	notesGroup := protectedGroup.Group("/notes")
	{
		notesGroup.GET("/shared", h.GetSharedNotes)
		notesGroup.GET("/user/:userId", h.GetUserNotes)
		notesGroup.DELETE("/:id/share/:userId", h.RemoveNoteShare)
		notesGroup.POST("/:id/share", h.ShareNote)
		notesGroup.GET("/:id", h.GetNote)
		notesGroup.PUT("/:id", h.UpdateNote)
		notesGroup.POST("", h.CreateNote)
		notesGroup.DELETE("/:id", h.DeleteNote)
	}

	foldersGroup := protectedGroup.Group("/folders")
	{
		foldersGroup.GET("/user/:userId", h.GetUserFolders)
		foldersGroup.POST("", h.CreateFolder)
		foldersGroup.DELETE("/:id", h.DeleteFolder)
	}
}

func (h *AssetHandler) GetNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	note, err := h.service.GetNoteByID(c.Request.Context(), requesterID, noteID)
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

	// Include access list if requester is owner (service will enforce owner-only)
	shares := []map[string]interface{}{}
	accessList, err := h.service.GetNoteAccess(c.Request.Context(), requesterID, noteID)
	if err == nil {
		shares = accessList
	}

	c.JSON(http.StatusOK, gin.H{"note": note, "shares": shares})
}

func (h *AssetHandler) CreateNote(c *gin.Context) {
	var req models.NoteCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": appErr.Message, "details": appErr.Details})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	note, err := h.service.CreateNote(c.Request.Context(), requesterID, req.Title, req.Content, req.FolderID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			if appErr.Type == utils.ErrTypeInternal {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": utils.ErrTypeInternalServer})
			} else {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message, "details": appErr.Details})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Note created successfully", "note": note})
}

func (h *AssetHandler) GetUserNotes(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	if userID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": utils.ErrTypeForbidden})
		return
	}

	notes, err := h.service.GetUserNotes(c.Request.Context(), requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *AssetHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	var req models.NoteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": appErr.Message, "details": appErr.Details})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	note, err := h.service.UpdateNote(c.Request.Context(), requesterID, noteID, req.Title, req.Content, req.FolderID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			if appErr.Type == utils.ErrTypeInternal {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": utils.ErrTypeInternalServer})
			} else {
				c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message, "details": appErr.Details})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note updated successfully", "note": note})
}

func (h *AssetHandler) DeleteNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	err = h.service.DeleteNote(c.Request.Context(), requesterID, noteID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note deleted successfully"})
}

func (h *AssetHandler) ShareNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	var req models.NoteShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": appErr.Message, "details": appErr.Details})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	err = h.service.ShareNote(c.Request.Context(), requesterID, noteID, req.UserID, req.Permission)
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

	c.JSON(http.StatusOK, gin.H{"message": "Note shared successfully"})
}

func (h *AssetHandler) RemoveNoteShare(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	err = h.service.RemoveNoteShare(c.Request.Context(), requesterID, noteID, userID)
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

	c.JSON(http.StatusOK, gin.H{"message": "Share removed successfully"})
}

func (h *AssetHandler) GetSharedNotes(c *gin.Context) {
	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	notes, err := h.service.GetSharedNotes(c.Request.Context(), requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *AssetHandler) CreateFolder(c *gin.Context) {
	var req models.FolderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := utils.FormatValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": appErr.Message, "details": appErr.Details})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	folder, err := h.service.CreateFolder(c.Request.Context(), requesterID, req.Name)
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

	c.JSON(http.StatusCreated, gin.H{"message": "Folder created successfully", "folder": folder})
}

func (h *AssetHandler) GetUserFolders(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	if userID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": utils.ErrTypeForbidden})
		return
	}

	folders, err := h.service.GetUserFolders(c.Request.Context(), requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folders": folders})
}

func (h *AssetHandler) DeleteFolder(c *gin.Context) {
	folderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrTypeInvalidId})
		return
	}

	requesterID, ok := getRequesterID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrTypeUnauthorized})
		return
	}

	err = h.service.DeleteFolder(c.Request.Context(), requesterID, folderID)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			c.JSON(utils.MapErrorToHTTPStatus(err), gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrTypeInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder deleted successfully"})
}
