package utils

import (
	"embed"
)

var (
	//go:embed templates/*.html
	templates embed.FS
)
