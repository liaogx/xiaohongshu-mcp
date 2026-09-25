package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liaogx/xiaohongshu-mcp/internal/sessioncmd"
)

func TestCLIDispatch(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		args          []string
		server        serverOptions
		recover       sessioncmd.RecoverOptions
	}{
		{"default", "serve", nil, serverOptions{headless: true, port: ":18060"}, sessioncmd.RecoverOptions{}},
		{"legacy flags", "serve", []string{"-headless=false", "-port=127.0.0.1:18061", "-token=fixture"}, serverOptions{port: "127.0.0.1:18061", token: "fixture"}, sessioncmd.RecoverOptions{}},
		{"explicit server", "serve", []string{"serve"}, serverOptions{headless: true, port: ":18060"}, sessioncmd.RecoverOptions{}},
		{"login", "login", []string{"login"}, serverOptions{}, sessioncmd.RecoverOptions{}},
		{"recover", "recover", []string{"recover"}, serverOptions{}, sessioncmd.RecoverOptions{Keyword: "咖啡"}},
		{"fresh", "recover", []string{"recover", "-fresh", "-keyword", "旅行"}, serverOptions{}, sessioncmd.RecoverOptions{Keyword: "旅行", Fresh: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			checkCommand := func(name string) {
				calls++
				if name != tc.command {
					t.Errorf("called %s, want %s", name, tc.command)
				}
			}
			actions := cliActions{
				serve: func(options serverOptions) error {
					checkCommand("serve")
					if options != tc.server {
						t.Error("server options mismatch")
					}
					return nil
				},
				login: func() error { checkCommand("login"); return nil },
				recover: func(options sessioncmd.RecoverOptions) error {
					checkCommand("recover")
					if options != tc.recover {
						t.Error("recovery options mismatch")
					}
					return nil
				},
			}
			var output bytes.Buffer
			if code := runCLI(tc.args, &output, &output, actions); code != 0 || calls != 1 {
				t.Fatalf("code=%d calls=%d output=%s", code, calls, output.String())
			}
		})
	}
}

func TestCLIReadOnlyCommandsAndInvalidArgumentsDoNotDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "must-not-exist.json")
	t.Setenv("COOKIES_PATH", path)
	t.Setenv("XHS_SITE", "invalid-for-help-and-version")
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"-version"}, 0, "xiaohongshu-mcp " + version},
		{[]string{"login", "-version"}, 0, "xiaohongshu-mcp " + version},
		{[]string{"recover", "-version"}, 0, "xiaohongshu-mcp " + version},
		{[]string{"-help"}, 0, "recover -fresh"},
		{[]string{"login", "-help"}, 0, "当前 login"},
		{[]string{"recover", "-help"}, 0, "-keyword"},
		{[]string{"unknown"}, 2, "未知子命令"},
		{[]string{"-headless=invalid"}, 2, "invalid"},
		{[]string{"login", "-fresh"}, 2, "flag provided but not defined"},
		{[]string{"-fresh"}, 2, "flag provided but not defined"},
		{[]string{"-headless=false", "login"}, 2, "子命令必须"},
		{[]string{"recover", "extra"}, 2, "多余参数"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var output bytes.Buffer
			// Nil handlers panic if a read-only or invalid command starts work.
			code := runCLI(tc.args, &output, &output, cliActions{})
			if code != tc.code || !strings.Contains(output.String(), tc.want) {
				t.Fatalf("code=%d output=%s", code, output.String())
			}
		})
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("read-only command created session output: %v", err)
	}
}

func TestCLIActionFailuresReturnNonzero(t *testing.T) {
	failure := errors.New("synthetic action failure")
	actions := cliActions{
		serve:   func(serverOptions) error { return failure },
		login:   func() error { return failure },
		recover: func(sessioncmd.RecoverOptions) error { return failure },
	}
	for _, command := range []string{"serve", "login", "recover"} {
		var output bytes.Buffer
		if code := runCLI([]string{command}, &output, &output, actions); code != 1 {
			t.Errorf("%s failure returned %d", command, code)
		}
	}
}
