//go:build embed

// Package embed holds embedded files for the application.
package embed

import "embed"

//go:embed all:../../apps/web/dist/*
var Assets embed.FS
