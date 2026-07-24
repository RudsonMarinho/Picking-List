// Package web embute a UI (HTML de origem, preservado — Fase 1 não usa
// Vercel) diretamente no binário via //go:embed.
package web

import _ "embed"

//go:embed index.html
var IndexHTML []byte
