package router

import (
	"github.com/gin-gonic/gin"
	"github.com/afshintalebi/rag-assistant/internal/api"
)

// SetupRouter configures the Gin engine and API routes.
func SetupRouter(handler *api.Handler) *gin.Engine {
	r := gin.Default()

	r.MaxMultipartMemory = 8 << 20 // 8 MB file upload limit

	// Serve the static frontend
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	apiGroup := r.Group("/api/v1")
	{
		apiGroup.POST("/chat", handler.HandleChat) // Now supports SSE Streaming
		apiGroup.POST("/scrape", handler.HandleScrape)
		apiGroup.POST("/upload", handler.HandleUploadPDF)
	}

	return r
}