package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testdata = "../../testdata"

// run executes the program with a clean environment. stdout is a bytes.Buffer,
// not a terminal, so the app takes the plain-output path.
func run(t *testing.T, env map[string]string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Setenv("PRETTY_ALIAS_DEBUG", "")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("SHELL", "")
	for k, v := range env {
		t.Setenv(k, v)
	}
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb, "test")
	return code, out.String(), errb.String()
}

func TestPlainOutputFromFixtures(t *testing.T) {
	tests := []struct {
		file, shell string
		wantLines   int
		wantRow     string
	}{
		{"bashrc", "bash", 168, "gs\tgit status -sb"},
		{"zshrc", "zsh", 157, "open\txdg-open"},
		{"config.fish", "fish", 103, "gs\tgit status -sb"},
		{"large/bashrc", "bash", 2000, ""},
		{"large/zshrc", "zsh", 2000, ""},
		{"large/config.fish", "fish", 2000, ""},
		{"minimal/bashrc", "bash", 1, "ll\tls -la"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			code, out, errs := run(t, nil, "--shell", tt.shell, "--file", filepath.Join(testdata, tt.file))
			if code != ExitOK {
				t.Fatalf("exit %d, stderr: %s", code, errs)
			}
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("%d lines, want %d", len(lines), tt.wantLines)
			}
			if tt.wantRow != "" && !contains(lines, tt.wantRow) {
				t.Errorf("missing row %q", tt.wantRow)
			}
			for _, l := range lines {
				if strings.Count(l, "\t") != 1 {
					t.Fatalf("line %q does not have exactly one tab", l)
				}
			}
		})
	}
}

func TestPlainFlagForcesPlain(t *testing.T) {
	code, out, _ := run(t, nil, "--plain", "--shell", "bash", "--file", filepath.Join(testdata, "minimal/bashrc"))
	if code != ExitOK || out != "ll\tls -la\n" {
		t.Errorf("code %d, out %q", code, out)
	}
}

func TestEveryFixtureRunsWithoutCrashing(t *testing.T) {
	// The whole point of the rewrite: no input file may make the program panic.
	shells := map[string]string{"zshrc": "zsh", "fish": "fish"}
	err := filepath.WalkDir(testdata, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(path, ".md") || strings.Contains(path, "/gen/") {
			return err
		}
		sh := "bash"
		for sub, s := range shells {
			if strings.Contains(path, sub) {
				sh = s
			}
		}
		t.Run(strings.TrimPrefix(path, testdata+"/"), func(t *testing.T) {
			code, out, errs := run(t, nil, "--shell", sh, "--file", path)
			if code != ExitOK {
				t.Errorf("exit %d: %s", code, errs)
			}
			if strings.ContainsAny(out, "\x1b\x07\r") {
				t.Errorf("control characters in output: %q", out)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoAliasesIsNotAnError(t *testing.T) {
	code, out, errs := run(t, nil, "--shell", "bash", "--file", filepath.Join(testdata, "empty/bashrc"))
	if code != ExitOK {
		t.Errorf("exit %d, want 0", code)
	}
	if out != "" {
		t.Errorf("stdout should be empty, got %q", out)
	}
	if !strings.Contains(errs, "no aliases found") || !strings.Contains(errs, "empty/bashrc") {
		t.Errorf("stderr should say where it looked: %q", errs)
	}
}

func TestErrorsAreReportedOnStderr(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	tests := []struct {
		name     string
		env      map[string]string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"missing file", nil, []string{"--shell", "bash", "--file", missing}, ExitError, "no such file"},
		{"unsupported $SHELL", map[string]string{"SHELL": "/bin/sh"}, []string{"--file", missing}, ExitError, "unsupported shell"},
		{"empty $SHELL", nil, []string{"--file", missing}, ExitError, "--shell"},
		{"bad --shell", nil, []string{"--shell", "nu", "--file", missing}, ExitError, "unsupported shell"},
		{"home without rc", map[string]string{"SHELL": "/bin/bash", "HOME": t.TempDir()}, nil, ExitError, ".bashrc"},
		{"unknown flag", nil, []string{"--nope"}, ExitUsage, "flag provided but not defined"},
		{"stray argument", nil, []string{"extra"}, ExitUsage, "unexpected argument"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errs := run(t, tt.env, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d (stderr %q)", code, tt.wantCode, errs)
			}
			if out != "" {
				t.Errorf("stdout should be empty: %q", out)
			}
			if !strings.Contains(errs, tt.wantErr) {
				t.Errorf("stderr %q does not contain %q", errs, tt.wantErr)
			}
		})
	}
}

func TestDetectsShellAndFileFromEnvironment(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("alias hi='echo hello'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(t, map[string]string{"HOME": home, "SHELL": "/usr/bin/zsh"})
	if code != ExitOK || out != "hi\techo hello\n" {
		t.Errorf("code %d out %q err %q", code, out, errs)
	}
}

func TestHelpAndVersion(t *testing.T) {
	code, out, _ := run(t, nil, "--version")
	if code != ExitOK || out != "pretty-alias test\n" {
		t.Errorf("--version: code %d out %q", code, out)
	}
	code, _, errs := run(t, nil, "-h")
	if code != ExitOK || !strings.Contains(errs, "--shell") || !strings.Contains(errs, "Exit codes") {
		t.Errorf("-h: code %d, usage %q", code, errs)
	}
}

func TestVersionFallback(t *testing.T) {
	if got := Version("v1.2.3"); got != "v1.2.3" {
		t.Errorf("injected version ignored: %q", got)
	}
	if got := Version("dev"); got == "" {
		t.Error("empty version")
	}
}

func TestDebugLogGoesToCacheDirNotCWD(t *testing.T) {
	cache := t.TempDir()
	cwd := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	file, _ := filepath.Abs(filepath.Join(old, testdata, "minimal/bashrc"))
	// Linux honours XDG_CACHE_HOME; macOS uses $HOME/Library/Caches. Point both
	// at the temp dir and ask the app where it will write.
	code, _, errs := run(t, map[string]string{"PRETTY_ALIAS_DEBUG": "1", "XDG_CACHE_HOME": cache, "HOME": cache},
		"--shell", "bash", "--file", file)
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	logPath, err := LogPath() // env set by run() is still in effect
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(logPath, cache) {
		t.Fatalf("log path %q is outside the test cache dir %q", logPath, cache)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log not written to cache dir: %v", err)
	}
	if !strings.Contains(string(b), "parsed") {
		t.Errorf("log content: %q", b)
	}
	if entries, _ := os.ReadDir(cwd); len(entries) != 0 {
		t.Errorf("files were written into the working directory: %v", entries)
	}
}

func TestTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	if got := tilde(filepath.Join(home, ".zshrc")); got != filepath.Join("~", ".zshrc") {
		t.Errorf("got %q", got)
	}
	if got := tilde("/etc/zshrc"); got != "/etc/zshrc" {
		t.Errorf("got %q", got)
	}
}

func contains(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}
