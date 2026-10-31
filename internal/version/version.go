// Package version exposes build metadata injected at link time.
package version

// These values are overridden through -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
)
