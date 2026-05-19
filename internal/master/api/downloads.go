package api

import (
	_ "embed"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterDownloadRoutes(r *gin.Engine, dataDir string) {
	r.GET("/install.sh", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/x-shellscript; charset=utf-8", []byte(agentInstallScript))
	})
	r.GET("/download/:name", func(c *gin.Context) {
		name := c.Param("name")
		if !allowedAgentDownloadName(name) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "not found"})
			return
		}
		c.File(filepath.Join(dataDir, "downloads", name))
	})
}

func allowedAgentDownloadName(name string) bool {
	return name == "agent-linux-amd64" || name == "agent-linux-arm64" ||
		strings.HasPrefix(name, "agent-linux-amd64.") ||
		strings.HasPrefix(name, "agent-linux-arm64.")
}

//go:embed assets/install.sh
var agentInstallScript string
