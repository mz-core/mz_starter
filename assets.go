package starter

import "embed"

//go:embed catalog/resources.json web/dist
var Assets embed.FS
