package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestSummary(t *testing.T) {
	got := Summary("example-tool", DefaultVersion)
	for _, want := range []string{"example-tool " + DefaultVersion, "Commit: " + Commit,
		"Build time (UTC): " + BuildTime, "Go: " + runtime.Version(),
		"Platform: " + runtime.GOOS + "/" + runtime.GOARCH} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in version output", want)
		}
	}
}
