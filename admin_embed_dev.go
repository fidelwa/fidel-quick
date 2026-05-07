//go:build !prod

package main

import "io/fs"

// adminFS en dev devuelve nil — el SPA se sirve por Vite en :5173.
// La versión productiva (build tag `prod`) embebe admin/dist.
func adminFS() fs.FS { return nil }
