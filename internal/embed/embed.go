package embed

import "embed"

//go:embed apps/web/dist/*
var Assets embed.FS
