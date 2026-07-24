// Package migrations embute os arquivos .sql no binário — o modo
// `-migrate` do invtech-api os aplica via golang-migrate, sem depender de
// arquivos soltos no filesystem da imagem distroless.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
