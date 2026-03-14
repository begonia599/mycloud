package handlers

import (
	"net/http"
	"strconv"

	"clouddisk/middleware"

	"github.com/begonia599/myplatform/sdk"
	"github.com/gin-gonic/gin"
)

// ImageHandler proxies image operations to the myplatform imagebed service via SDK.
type ImageHandler struct {
	Platform *sdk.Client
}

// Upload proxies image upload to myplatform
func (h *ImageHandler) Upload(c *gin.Context) {
	token := middleware.CurrentToken(c)

	fh, err := c.FormFile("images")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing images field"})
		return
	}

	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	scoped := h.Platform.WithToken(token)
	img, err := scoped.ImageBed.UploadReader(fh.Filename, src)
	if err != nil {
		status := http.StatusInternalServerError
		if apiErr, ok := err.(*sdk.APIError); ok {
			status = apiErr.StatusCode
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"images": []sdk.Image{*img}})
}

// List proxies image listing to myplatform
func (h *ImageHandler) List(c *gin.Context) {
	token := middleware.CurrentToken(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	scoped := h.Platform.WithToken(token)
	resp, err := scoped.ImageBed.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": resp.Data,
		"total":  resp.Total,
	})
}

// Delete proxies image deletion to myplatform
func (h *ImageHandler) Delete(c *gin.Context) {
	token := middleware.CurrentToken(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	scoped := h.Platform.WithToken(token)
	if err := scoped.ImageBed.Delete(uint(id)); err != nil {
		status := http.StatusInternalServerError
		if apiErr, ok := err.(*sdk.APIError); ok {
			status = apiErr.StatusCode
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "image deleted"})
}

// ToggleVisibility proxies visibility change to myplatform
func (h *ImageHandler) ToggleVisibility(c *gin.Context) {
	token := middleware.CurrentToken(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	scoped := h.Platform.WithToken(token)
	img, err := scoped.ImageBed.ToggleVisibility(uint(id), req.IsPublic)
	if err != nil {
		status := http.StatusInternalServerError
		if apiErr, ok := err.(*sdk.APIError); ok {
			status = apiErr.StatusCode
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"image": img})
}

// PlatformURL returns the base URL of the platform for frontend to construct public image URLs
func (h *ImageHandler) PlatformURL(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"platform_url": h.Platform.GetBaseURL()})
}
