package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIKeyMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-KEY")

		if key == "" || key != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid api key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
