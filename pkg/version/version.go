// Package version exposes the service version. Populated via -ldflags -X at
// build time; falls back to runtime/debug.ReadBuildInfo otherwise.
package version

import "runtime/debug"

// Version is the running service version, populated at build time via
// -ldflags "-X restful-boilerplate/pkg/version.Version=...". Falls back to
// runtime build info, then "dev".
//
//nolint:gochecknoglobals // populated via -ldflags -X at build time
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		Version = info.Main.Version
	}
}
