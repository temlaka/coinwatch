package docs

import (
	"embed"
	"net/http"
)

//go:embed openapi.yaml
var files embed.FS

func SpecHandler() http.Handler {
	return http.FileServerFS(files)
}
