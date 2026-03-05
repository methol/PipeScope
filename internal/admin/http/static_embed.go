package http

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

func EmbeddedStaticFS() fs.FS {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return staticFiles
	}
	return sub
}

