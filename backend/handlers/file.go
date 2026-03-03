package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

// uploadMeta stores metadata for a chunked upload session.
type uploadMeta struct {
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	ChunkSize  int64  `json:"chunk_size"`
	TotalChunks int   `json:"total_chunks"`
}

func (h *FileHandler) tmpDir() string {
	return filepath.Join(h.Config.UploadDir, "tmp")
}

// InitUpload creates an upload session: generates upload_id, creates temp dir, writes .meta.json.
func (h *FileHandler) InitUpload(c *gin.Context) {
	var req struct {
		FileName  string `json:"file_name" binding:"required"`
		FileSize  int64  `json:"file_size" binding:"required"`
		ChunkSize int64  `json:"chunk_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.ChunkSize <= 0 {
		req.ChunkSize = 4 * 1024 * 1024 // default 4MB
	}

	totalChunks := int(req.FileSize / req.ChunkSize)
	if req.FileSize%req.ChunkSize != 0 {
		totalChunks++
	}

	uploadID := uuid.New().String()
	dir := filepath.Join(h.tmpDir(), uploadID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp directory"})
		return
	}

	meta := uploadMeta{
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
	}
	metaBytes, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, ".meta.json"), metaBytes, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id":    uploadID,
		"chunk_size":   req.ChunkSize,
		"total_chunks": totalChunks,
	})
}

// UploadChunk receives a single chunk and saves it to the temp directory.
func (h *FileHandler) UploadChunk(c *gin.Context) {
	uploadID := c.PostForm("upload_id")
	if _, err := uuid.Parse(uploadID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload_id"})
		return
	}

	chunkIndexStr := c.PostForm("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}

	dir := filepath.Join(h.tmpDir(), uploadID)
	// Verify upload session exists
	if _, err := os.Stat(filepath.Join(dir, ".meta.json")); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
		return
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing chunk data"})
		return
	}
	defer file.Close()

	dst, err := os.Create(filepath.Join(dir, fmt.Sprintf("%d", chunkIndex)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write chunk"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunk_index": chunkIndex})
}

// UploadStatus returns the list of uploaded chunk indices for resumable uploads.
func (h *FileHandler) UploadStatus(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if _, err := uuid.Parse(uploadID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload_id"})
		return
	}

	dir := filepath.Join(h.tmpDir(), uploadID)
	metaPath := filepath.Join(dir, ".meta.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
		return
	}

	var meta uploadMeta
	json.Unmarshal(metaBytes, &meta)

	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read upload directory"})
		return
	}

	var uploaded []int
	for _, e := range entries {
		if e.Name() == ".meta.json" {
			continue
		}
		idx, err := strconv.Atoi(e.Name())
		if err == nil {
			uploaded = append(uploaded, idx)
		}
	}
	sort.Ints(uploaded)

	c.JSON(http.StatusOK, gin.H{
		"upload_id":    uploadID,
		"file_name":    meta.FileName,
		"file_size":    meta.FileSize,
		"chunk_size":   meta.ChunkSize,
		"total_chunks": meta.TotalChunks,
		"uploaded":     uploaded,
	})
}

// CompleteUpload merges all chunks into the final file, creates a DB record, and cleans up.
func (h *FileHandler) CompleteUpload(c *gin.Context) {
	var req struct {
		UploadID string `json:"upload_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if _, err := uuid.Parse(req.UploadID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload_id"})
		return
	}

	dir := filepath.Join(h.tmpDir(), req.UploadID)
	metaPath := filepath.Join(dir, ".meta.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
		return
	}

	var meta uploadMeta
	json.Unmarshal(metaBytes, &meta)

	// Verify all chunks exist
	for i := 0; i < meta.TotalChunks; i++ {
		chunkPath := filepath.Join(dir, fmt.Sprintf("%d", i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("missing chunk %d", i)})
			return
		}
	}

	// Merge chunks into final file
	if err := os.MkdirAll(h.Config.UploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}

	ext := filepath.Ext(meta.FileName)
	storedName := uuid.New().String() + ext
	finalPath := filepath.Join(h.Config.UploadDir, storedName)

	outFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output file"})
		return
	}
	defer outFile.Close()

	var totalSize int64
	for i := 0; i < meta.TotalChunks; i++ {
		chunkPath := filepath.Join(dir, fmt.Sprintf("%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read chunk %d", i)})
			return
		}
		n, err := io.Copy(outFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge chunks"})
			return
		}
		totalSize += n
	}

	// Detect MIME type from extension
	mimeType := "application/octet-stream"
	extLower := strings.ToLower(ext)
	mimeMap := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
		".pdf": "application/pdf", ".zip": "application/zip",
		".mp4": "video/mp4", ".mp3": "audio/mpeg", ".wav": "audio/wav",
		".txt": "text/plain", ".html": "text/html", ".css": "text/css",
		".js": "application/javascript", ".json": "application/json",
		".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls": "application/vnd.ms-excel", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
	if m, ok := mimeMap[extLower]; ok {
		mimeType = m
	}

	fileRecord := models.File{
		Name:       meta.FileName,
		StoredName: storedName,
		Size:       totalSize,
		MimeType:   mimeType,
	}
	if err := database.DB.Create(&fileRecord).Error; err != nil {
		os.Remove(finalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file record"})
		return
	}

	// Clean up temp directory
	os.RemoveAll(dir)

	c.JSON(http.StatusOK, gin.H{"file": fileRecord})
}
