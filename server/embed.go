// Package tracker provides the embedded web application filesystem.
package tracker

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:web
var webFS embed.FS

// WebFS returns an fs.FS rooted at the web directory.
func WebFS() fs.FS {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed: failed to create sub filesystem: %v", err)
	}
	return sub
}
