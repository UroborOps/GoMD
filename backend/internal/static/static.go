package static

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// FS returns the embedded frontend/dist contents.
func FS() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}

// ReadFile reads a file from the embedded dist directory.
func ReadFile(name string) ([]byte, error) {
	return files.ReadFile("dist/" + name)
}
