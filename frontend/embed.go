package frontend

import "embed"

// FS holds the compiled frontend assets.
// Run "bun run build" in the frontend/ directory before
// building the Go binary to populate the dist/ folder.
//
//go:embed all:dist
var FS embed.FS
