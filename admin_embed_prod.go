//go:build prod

package main

import (
	"embed"
	"io/fs"
)

// adminDist es el bundle compilado de admin/ que el Dockerfile copia
// desde el stage admin-builder antes de `go build -tags prod`. En dev
// (sin el tag) se usa admin_embed_dev.go que devuelve nil.
//
//go:embed all:admin/dist
var adminDist embed.FS

// adminFS devuelve el filesystem del SPA admin para servirlo desde el
// binario en producción. En dev devuelve nil (Vite sirve en :5173).
func adminFS() fs.FS {
	sub, err := fs.Sub(adminDist, "admin/dist")
	if err != nil {
		// Embed mal armado; preferible fallar al inicio.
		panic("admin SPA embed: " + err.Error())
	}
	return sub
}
