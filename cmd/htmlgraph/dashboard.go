package main

import (
	"embed"
	"io/fs"
)

//go:embed dashboard/*
var dashboardFS embed.FS

// dashboardSub returns the dashboard/ subdirectory as an fs.FS,
// so files are served at / instead of /dashboard/.
func dashboardSub() fs.FS {
	sub, _ := fs.Sub(dashboardFS, "dashboard")
	return sub
}
