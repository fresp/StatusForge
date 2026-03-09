package embed

import "embed"

//go:embed all:web/dist
var Assets embed.FS
