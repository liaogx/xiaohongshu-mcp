package sessioncmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidSiteDoesNotStartSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "must-not-exist.json")
	t.Setenv("COOKIES_PATH", path)
	t.Setenv("XHS_SITE", "invalid-fixture")
	if err := Login(io.Discard); err == nil {
		t.Fatal("login accepted invalid site")
	}
	if err := Recover(RecoverOptions{Fresh: true}, io.Discard); err == nil {
		t.Fatal("recovery accepted invalid site")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("invalid configuration changed session output: %v", err)
	}
}

func TestRecoveryTimeoutIsFailure(t *testing.T) {
	if err := waitForRecovery(nil, false, io.Discard, 0); err == nil {
		t.Fatal("timeout reported success")
	}
}
