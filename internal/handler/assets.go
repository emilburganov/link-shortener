package handler

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed assets/*.woff2
var assetsFS embed.FS

func (h *Handler) asset(c *gin.Context) {
	data, err := assetsFS.ReadFile("assets/" + c.Param("name"))

	if err != nil {
		c.Status(http.StatusNotFound)

		return
	}

	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Data(http.StatusOK, "font/woff2", data)
}
