package ui

import (
	"embed"
	"io/fs"
)

//go:embed assets
var embeddedFiles embed.FS

// Files exports the sub-directory so "assets" isn't part of the path
var Files, _ = fs.Sub(embeddedFiles, "assets")
