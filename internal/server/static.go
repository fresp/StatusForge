// Package server provides static file serving.
package server

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"status-platform/internal/embed" // Import our embeddings
)

// StaticFileServer serves static files from embedded FS
func StaticFileServer() gin.HandlerFunc {
	fSys, err := fs.Sub(embed.Assets, "web/dist")
	if err != nil {
		log.Printf("[STATIC] Error creating sub filesystem: %v", err)
		return func(c *gin.Context) {
			c.String(http.StatusNotFound, "Embedded files not available")
		}
	}

	fileServer := http.FileServer(http.FS(fSys))

	return func(c *gin.Context) {

		path := c.Request.URL.Path

		// HANDLE ROOT DIRECTLY
		if path == "/" {
			c.FileFromFS("index.html", http.FS(fSys))
			c.Abort()
			return
		}

		// CHECK FILE EXISTENCE
		_, err := fSys.Open(path[1:])
		if err != nil {
			// SPA fallback
			c.FileFromFS("index.html", http.FS(fSys))
			c.Abort()
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}