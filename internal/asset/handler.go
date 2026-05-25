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

// GetNote godoc
// @Summary      Get a note by ID
// @Description  Retrieves a note by its ID. Requires read or write access.
// @Tags         Notes
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id} [get]
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

// GetNoteAccess godoc
// @Summary      Get note access list
// @Description  Retrieves the list of users who have access to a note and their permissions. Only the owner can view this.
// @Tags         Notes
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id}/access [get]
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

// GetUserNotes godoc
// @Summary      Get notes of a user
// @Description  Retrieves all notes that belong to a user. User can only access their own notes.
// @Tags         Notes
// @Produce      json
// @Param        userId   path      int  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Router       /notes/user/{userId} [get]
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

// UpdateNote godoc
// @Summary      Update a note
// @Description  Updates the title, content, or folder of a note. Requires write access.
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Param        request  body      map[string]interface{}  true  "Example: {'title': 'New Title', 'content': 'Updated content', 'folder_id': 2}"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id} [put]
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

// DeleteNote godoc
// @Summary      Delete a note
// @Description  Deletes a note by its ID. Requires owner or write access.
// @Tags         Notes
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id} [delete]
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

// ShareNote godoc
// @Summary      Share a note with another user
// @Description  Shares a note with another user by granting them read or write access. Requires owner or write access.
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Param        request  body      map[string]interface{}  true  "Example: {'user_id': 2, 'permission': 'read'}"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id}/share [post]
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

// RemoveNoteShare godoc
// @Summary      Remove a user's access to a note
// @Description  Removes a user's access to a note. Requires owner or write access.
// @Tags         Notes
// @Produce      json
// @Param        id   path      int  true  "Note ID"
// @Param        userId   path      int  true  "User ID to remove access for"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id}/share/{userId} [delete]
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

// CreateNote godoc
// @Summary      Create a new note
// @Description  Creates a new note with a title, content, and optional folder. The note will be owned by the requester.
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]interface{}  true  "Example: {'title': 'Note Title', 'content': 'Note content', 'folder_id': 2}"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /notes [post]
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

// CreateFolder godoc
// @Summary      Create a new folder
// @Description  Creates a new folder with a specified name. The folder will be owned by the requester.
// @Tags         Folders
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Example: {'name': 'Project Documents'}"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /folders [post]
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

// GetUserFolders godoc
// @Summary      Get folders of a user
// @Description  Retrieves all folders that belong to a user. User can only access their own folders.
// @Tags         Folders
// @Produce      json
// @Param        userId   path      int  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Router       /folders/user/{userId} [get]
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

// DeleteFolder godoc
// @Summary      Delete a folder
// @Description  Deletes a folder by its ID. Only the owner can delete a folder.
// @Tags         Folders
// @Produce      json
// @Param        id   path      int  true  "Folder ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /folders/{id} [delete]
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
