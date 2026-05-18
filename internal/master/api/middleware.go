package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vaultfleet/internal/master/db"
)

func RequireAuth(sessions *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
			return
		}

		session, ok := sessions.Get(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
			return
		}

		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)
		c.Next()
	}
}

func RequireInit(database *db.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var count int64
		if err := database.DB.Model(&db.User{}).Count(&count).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
			return
		}
		if count == 0 {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"ok": false, "error": "init_required"})
			return
		}

		c.Next()
	}
}
