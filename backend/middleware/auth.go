package middleware

import (
	"net/http"
	"strings"

	"github.com/begonia599/myplatform/sdk"
	"github.com/gin-gonic/gin"
)

const (
	contextUserKey  = "user"
	contextTokenKey = "token"
)

// AuthRequired 通过统一后端 SDK 验证 token
func AuthRequired(platform *sdk.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		token := parts[1]

		result, err := platform.Auth.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token verification failed"})
			return
		}
		if !result.Valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(contextUserKey, result.User)
		c.Set(contextTokenKey, token)
		c.Next()
	}
}

// CurrentUser 从 context 中获取已验证的用户信息
func CurrentUser(c *gin.Context) (sdk.VerifyUser, bool) {
	v, exists := c.Get(contextUserKey)
	if !exists {
		return sdk.VerifyUser{}, false
	}
	u, ok := v.(sdk.VerifyUser)
	return u, ok
}

// CurrentToken 从 context 中获取原始 token
func CurrentToken(c *gin.Context) string {
	return c.GetString(contextTokenKey)
}

// RequirePermission 通过统一后端 SDK 检查用户权限
func RequirePermission(platform *sdk.Client, object, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		allowed, err := platform.Permission.CheckPermission(user.ID, object, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "permission check failed"})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
