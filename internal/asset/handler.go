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
	assetGroup := protectedGroup.Group("/notes")
	{
		assetGroup.GET("/:id", h.GetNote)
		assetGroup.PUT("/:id", h.UpdateNote)
	}
}

func (h *AssetHandler) GetNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	// Safely extract requester ID from the JWT Bouncer
	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	note, err := h.service.GetNoteByID(requesterID, noteID)
	if err != nil {
		if err.Error() == "forbidden: you do not have access to this asset" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"note": note})
}

func (h *AssetHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterCtx, _ := c.Get("userID")
	requesterID := int64(requesterCtx.(float64))

	note, err := h.service.UpdateNote(requesterID, noteID, req.Title, req.Content)
	if err != nil {
		if err.Error() == "forbidden: you do not have write access to this note" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update note"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "note updated successfully", "note": note})
}
