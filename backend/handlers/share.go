package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"clouddisk/config"
	"clouddisk/database"
	"clouddisk/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type ShareHandler struct {
	Config *config.Config
}

type CreateShareRequest struct {
	Title     string   `json:"title"`
	Password  string   `json:"password"`
	FileIDs   []string `json:"file_ids" binding:"required,min=1"`
	ExpiresIn *int     `json:"expires_in"` // hours
}

func generateCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *ShareHandler) Create(c *gin.Context) {
	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify all files exist
	var files []models.File
	if err := database.DB.Where("id IN ?", req.FileIDs).Find(&files).Error; err != nil || len(files) != len(req.FileIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "one or more files not found"})
		return
	}

	share := models.Share{
		Code:  generateCode(),
		Title: req.Title,
		Files: files,
	}

	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		share.Password = string(hashed)
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		share.ExpiresAt = &t
	}

	if err := database.DB.Create(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"share": gin.H{
			"id":         share.ID,
			"code":       share.Code,
			"title":      share.Title,
			"has_password": share.Password != "",
			"expires_at": share.ExpiresAt,
			"created_at": share.CreatedAt,
			"files":      share.Files,
		},
	})
}

func (h *ShareHandler) List(c *gin.Context) {
	var shares []models.Share
	if err := database.DB.Preload("Files").Order("created_at DESC").Find(&shares).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shares"})
		return
	}

	result := make([]gin.H, len(shares))
	for i, s := range shares {
		result[i] = gin.H{
			"id":           s.ID,
			"code":         s.Code,
			"title":        s.Title,
			"has_password": s.Password != "",
			"expires_at":   s.ExpiresAt,
			"created_at":   s.CreatedAt,
			"files":        s.Files,
		}
	}

	c.JSON(http.StatusOK, gin.H{"shares": result})
}

func (h *ShareHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var share models.Share
	if err := database.DB.First(&share, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	// Clear associations and delete
	database.DB.Where("share_id = ?", id).Delete(&models.ShareFile{})
	database.DB.Delete(&share)

	c.JSON(http.StatusOK, gin.H{"message": "share deleted"})
}

// --- Public endpoints ---

func (h *ShareHandler) GetShareInfo(c *gin.Context) {
	code := c.Param("code")

	var share models.Share
	if err := database.DB.Where("code = ?", code).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}

	resp := gin.H{
		"title":        share.Title,
		"has_password": share.Password != "",
	}

	// If no password, include files directly
	if share.Password == "" {
		database.DB.Preload("Files").Where("code = ?", code).First(&share)
		resp["files"] = share.Files
	}

	c.JSON(http.StatusOK, resp)
}

type VerifyRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *ShareHandler) VerifyShare(c *gin.Context) {
	code := c.Param("code")

	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	var share models.Share
	if err := database.DB.Preload("Files").Where("code = ?", code).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": share.Files})
}

func (h *ShareHandler) Download(c *gin.Context) {
	code := c.Param("code")
	fileID := c.Param("fileId")

	var share models.Share
	if err := database.DB.Preload("Files").Where("code = ?", code).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}

	// If share has password, verify via query param (simple approach for downloads)
	if share.Password != "" {
		pwd := c.Query("pwd")
		if pwd == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password required"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(pwd)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
			return
		}
	}

	// Check file belongs to this share
	var found *models.File
	for _, f := range share.Files {
		if f.ID == fileID {
			found = &f
			break
		}
	}
	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found in share"})
		return
	}

	filePath := filepath.Join(h.Config.UploadDir, found.StoredName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found on disk"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+found.Name+"\"")
	c.File(filePath)
}
