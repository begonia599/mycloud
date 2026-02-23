package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"clouddisk/config"
	"clouddisk/database"
	"clouddisk/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FileHandler struct {
	Config *config.Config
}

func (h *FileHandler) Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form data"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	if err := os.MkdirAll(h.Config.UploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}

	var saved []models.File
	for _, f := range files {
		storedName := uuid.New().String() + filepath.Ext(f.Filename)
		dst := filepath.Join(h.Config.UploadDir, storedName)

		if err := c.SaveUploadedFile(f, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %s", f.Filename)})
			return
		}

		fileRecord := models.File{
			Name:       f.Filename,
			StoredName: storedName,
			Size:       f.Size,
			MimeType:   f.Header.Get("Content-Type"),
		}
		if err := database.DB.Create(&fileRecord).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file record"})
			return
		}
		saved = append(saved, fileRecord)
	}

	c.JSON(http.StatusOK, gin.H{"files": saved})
}

func (h *FileHandler) List(c *gin.Context) {
	var files []models.File
	if err := database.DB.Order("created_at DESC").Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var file models.File
	if err := database.DB.First(&file, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Remove from any shares first
	database.DB.Where("file_id = ?", id).Delete(&models.ShareFile{})

	// Delete physical file
	os.Remove(filepath.Join(h.Config.UploadDir, file.StoredName))

	// Delete database record
	database.DB.Delete(&file)

	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}
