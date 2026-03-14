package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"clouddisk/config"
	"clouddisk/database"
	"clouddisk/middleware"
	"clouddisk/models"

	"github.com/begonia599/myplatform/sdk"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxImageSize = 10 * 1024 * 1024 // 10MB

// 允许的图片 MIME 类型
var allowedImageTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
	"image/bmp":     true,
	"image/x-icon":  true,
}

type ImageHandler struct {
	Config   *config.Config
	Platform *sdk.Client
}

func (h *ImageHandler) imageDir() string {
	return filepath.Join(h.Config.UploadDir, "images")
}

// Upload 上传图片（支持多文件）
func (h *ImageHandler) Upload(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form data"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no images provided"})
		return
	}

	dir := h.imageDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create image directory"})
		return
	}

	var saved []models.Image
	for _, f := range files {
		// 校验文件大小
		if f.Size > maxImageSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件 %s 超过 10MB 限制", f.Filename)})
			return
		}

		// 校验 MIME 类型
		mime := f.Header.Get("Content-Type")
		if !allowedImageTypes[mime] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件 %s 不是允许的图片类型", f.Filename)})
			return
		}

		storedName := uuid.New().String() + filepath.Ext(f.Filename)
		dst := filepath.Join(dir, storedName)

		if err := c.SaveUploadedFile(f, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save image: %s", f.Filename)})
			return
		}

		record := models.Image{
			UserID:     user.ID,
			Name:       f.Filename,
			StoredName: storedName,
			Size:       f.Size,
			MimeType:   mime,
			IsPublic:   true,
		}
		if err := database.DB.Create(&record).Error; err != nil {
			os.Remove(dst)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image record"})
			return
		}
		saved = append(saved, record)
	}

	// 构建包含直链的响应
	type imageResp struct {
		models.Image
		URL string `json:"url"`
	}
	resp := make([]imageResp, len(saved))
	for i, img := range saved {
		resp[i] = imageResp{Image: img, URL: fmt.Sprintf("/i/%s", img.ID)}
	}

	c.JSON(http.StatusOK, gin.H{"images": resp})
}

// List 列出当前用户的图片
func (h *ImageHandler) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var images []models.Image
	query := database.DB.Where("user_id = ?", user.ID).Order("created_at DESC")

	if err := query.Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list images"})
		return
	}

	type imageResp struct {
		models.Image
		URL string `json:"url"`
	}
	resp := make([]imageResp, len(images))
	for i, img := range images {
		resp[i] = imageResp{Image: img, URL: fmt.Sprintf("/i/%s", img.ID)}
	}

	c.JSON(http.StatusOK, gin.H{"images": resp})
}

// Delete 删除图片（仅限本人或 admin）
func (h *ImageHandler) Delete(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id := c.Param("id")

	var image models.Image
	if err := database.DB.First(&image, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// 检查权限：只能删自己的，admin 可以删任何人的
	if image.UserID != user.ID && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己的图片"})
		return
	}

	// 删除物理文件
	os.Remove(filepath.Join(h.imageDir(), image.StoredName))

	// 删除数据库记录
	database.DB.Delete(&image)

	c.JSON(http.StatusOK, gin.H{"message": "image deleted"})
}

// ToggleVisibility 切换公开/私有
func (h *ImageHandler) ToggleVisibility(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id := c.Param("id")

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var image models.Image
	if err := database.DB.First(&image, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// 只能修改自己的图片
	if image.UserID != user.ID && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能修改自己的图片"})
		return
	}

	image.IsPublic = req.IsPublic
	database.DB.Save(&image)

	c.JSON(http.StatusOK, gin.H{"image": image})
}

// Serve 公开访问图片
// 公开图片：无需认证，直接返回
// 私有图片：需要 Bearer token
func (h *ImageHandler) Serve(c *gin.Context) {
	id := c.Param("id")

	var image models.Image
	if err := database.DB.First(&image, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// 公开图片：直接返回
	if image.IsPublic {
		h.serveImage(c, &image)
		return
	}

	// 私有图片：验证 token
	header := c.GetHeader("Authorization")
	if header == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "此图片为私有，需要认证"})
		return
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
		return
	}

	result, err := h.Platform.Auth.Verify(parts[1])
	if err != nil || !result.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	h.serveImage(c, &image)
}

func (h *ImageHandler) serveImage(c *gin.Context, image *models.Image) {
	filePath := filepath.Join(h.imageDir(), image.StoredName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "image file not found on disk"})
		return
	}

	// 设置缓存头（公开图片缓存 7 天）
	if image.IsPublic {
		c.Header("Cache-Control", "public, max-age=604800")
	} else {
		c.Header("Cache-Control", "private, no-cache")
	}

	c.Header("Content-Type", image.MimeType)
	c.File(filePath)
}
