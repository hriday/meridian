// Package version holds the application's build version.
package version

// Version is the running build's version. It defaults to "dev" for local and
// unversioned builds, and is overridden at release time via
//
//	-ldflags "-X github.com/hriday/meridian/internal/version.Version=vX.Y.Z"
//
// (see .github/workflows/release.yml).
var Version = "dev"
