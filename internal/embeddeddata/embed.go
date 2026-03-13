package embeddeddata

import "embed"

//go:embed assets/*
var assetsFS embed.FS
