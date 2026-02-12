package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminRuntime 用于执行需要运行中服务的管理操作。
type AdminRuntime interface {
	KickPlayer(playerID string) bool
}

func (s *HTTPServer) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(s.adminToken) == "" {
			c.JSON(http.StatusInternalServerError, InternalErrorResponse("admin token not configured"))
			c.Abort()
			return
		}
		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" {
			token = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer"))
		}
		if token == "" || token != s.adminToken {
			c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "invalid admin token"))
			c.Abort()
			return
		}
		c.Next()
	}
}
