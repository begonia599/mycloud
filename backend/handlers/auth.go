package handlers

import (
	"net/http"

	"github.com/begonia599/myplatform/sdk"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	Platform *sdk.Client
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 代理到统一后端的登录接口
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tokens, err := h.Platform.Auth.Login(req.Username, req.Password)
	if err != nil {
		if apiErr, ok := err.(*sdk.APIError); ok {
			c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh 代理到统一后端的刷新接口
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 创建临时客户端来执行 refresh
	tmp := h.Platform.WithToken("")
	tmp.SetTokens("", req.RefreshToken, 0)
	tokens, err := tmp.Auth.Refresh()
	if err != nil {
		if apiErr, ok := err.(*sdk.APIError); ok {
			c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Message})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}
