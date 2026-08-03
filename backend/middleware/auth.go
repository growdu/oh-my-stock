package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"oh-my-stock/config"
)

// ============================================================
// 鉴权中间件 - 校验 Authorization: Bearer <token>
// 失败：401；成功：把 user_id 放入 ctx "user_id"
// ============================================================
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		var token string
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		} else {
			token = auth
		}
		uid, err := config.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: " + err.Error()})
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

// 便捷：直接从 ctx 取 user_id
func GetUserID(c *gin.Context) string {
	v, ok := c.Get("user_id")
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
