package server

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/fresp/Statora/internal/embed"
	"github.com/gin-gonic/gin"
)

func StaticFileServer() gin.HandlerFunc {

	fSys, err := fs.Sub(embed.Assets, "dist")
	if err != nil {
		log.Fatal(err)
	}

	return func(c *gin.Context) {

		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		if path == "" {
			path = "index.html"
		}

		log.Println("STATIC PATH:", path)

		file, err := fSys.Open(path)
		if err == nil {
			defer file.Close()

			stat, _ := file.Stat()

			http.ServeContent(
				c.Writer,
				c.Request,
				path,
				stat.ModTime(),
				file.(io.ReadSeeker),
			)
			return
		}

		// SPA fallback
		index, _ := fSys.Open("index.html")
		defer index.Close()

		stat, _ := index.Stat()

		http.ServeContent(
			c.Writer,
			c.Request,
			"index.html",
			stat.ModTime(),
			index.(io.ReadSeeker),
		)
	}
}
