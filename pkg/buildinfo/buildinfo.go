// Package buildinfo reports release metadata without accessing an account or browser.
package buildinfo

import (
	"fmt"
	"runtime"
)

// DefaultVersion is shared by source builds of all distributed executables.
const DefaultVersion = "1.0.0"

// Release builds set these fields using the Go linker's -X option.
var (
	Commit    = "unknown"
	BuildTime = "unknown"
)

func Summary(program, version string) string {
	return fmt.Sprintf("%s %s\nCommit: %s\nBuild time (UTC): %s\nGo: %s\nPlatform: %s/%s\n",
		program, version, Commit, BuildTime, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
