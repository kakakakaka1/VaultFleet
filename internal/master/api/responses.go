package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func writeErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"ok": false, "error": message})
}

func writeDataResponse(c *gin.Context, status int, data any) {
	body := gin.H{
		"ok":   true,
		"data": data,
	}

	raw, err := json.Marshal(data)
	if err == nil {
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err == nil {
			for key, value := range fields {
				body[key] = value
			}
		}
	}

	c.JSON(status, body)
}
