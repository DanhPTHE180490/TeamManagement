package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AssetHandler struct {
	service AssetService
}

func NewAssetHandler(service AssetService) *AssetHandler {
	return &AssetHandler{service: service}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	if requesterCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requesterCtx.(float64); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	requesterID := int64(requesterCtx.(float64))

	note, err := h.service.GetNoteByID(requesterID, noteID)
	if err != nil {
		if err.Error() == "access denied: you do not have permission to view this note" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"note": note})
}

func (h *AssetHandler) CreateNote(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		FolderID *int64 `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterCtx, _ := c.Get("userID")
	if requesterCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requesterCtx.(float64); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	requesterID := int64(requesterCtx.(float64))

	note, err := h.service.CreateNote(requesterID, req.Title, req.Content, req.FolderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create note"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "note created successfully", "note": note})
}

func (h *AssetHandler) GetUserNotes(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	// Users can only see their own notes
	if userID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	notes, err := h.service.GetUserNotes(requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *AssetHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		FolderID *int64 `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	note, err := h.service.UpdateNote(requesterID, noteID, req.Title, req.Content, req.FolderID)
	if err != nil {
		if err.Error() == "access denied: you do not have permission to edit this note" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update note"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "note updated successfully", "note": note})
}

func (h *AssetHandler) DeleteNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	err = h.service.DeleteNote(requesterID, noteID)
	if err != nil {
		if err.Error() == "access denied: only owner can delete this note" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete note"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "note deleted successfully"})
}

func (h *AssetHandler) ShareNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	var req struct {
		UserID     int64  `json:"user_id" binding:"required"`
		Permission string `json:"permission" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	err = h.service.ShareNote(requesterID, noteID, req.UserID, req.Permission)
	if err != nil {
		if err.Error() == "access denied: only owner can share this note" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "note shared successfully"})
}

func (h *AssetHandler) RemoveNoteShare(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	err = h.service.RemoveNoteShare(requesterID, noteID, userID)
	if err != nil {
		if err.Error() == "access denied: only owner can modify shares" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share removed successfully"})
}

func (h *AssetHandler) GetSharedNotes(c *gin.Context) {
	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	notes, err := h.service.GetSharedNotes(requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch shared notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *AssetHandler) CreateFolder(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	folder, err := h.service.CreateFolder(requesterID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "folder created successfully", "folder": folder})
}

func (h *AssetHandler) GetUserFolders(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	// Users can only see their own folders
	if userID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	folders, err := h.service.GetUserFolders(requesterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch folders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folders": folders})
}

func (h *AssetHandler) DeleteFolder(c *gin.Context) {
	folderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid folder ID"})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	err = h.service.DeleteFolder(requesterID, folderID)
	if err != nil {
		if err.Error() == "access denied: only owner can delete this folder" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete folder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "folder deleted successfully"})
}
