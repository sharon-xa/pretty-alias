package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// flagNames returns every flag listed by --help, so a new flag can't be left
// out of the completion scripts.
func flagNames(t *testing.T) []string {
	t.Helper()
	_, _, help := run(t, nil, "--help")
	var names []string
	for _, m := range regexp.MustCompile(`(?m)^  -(\w+)`).FindAllStringSubmatch(help, -1) {
		names = append(names, m[1])
	}
	if len(names) < 7 {
		t.Fatalf("found only %v in --help output:\n%s", names, help)
	}
	return append(names, "help")
}

func TestCompletionScriptsCoverEveryFlag(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish"} {
		t.Run(sh, func(t *testing.T) {
			code, out, errs := run(t, nil, "--completion", sh)
			if code != ExitOK || errs != "" {
				t.Fatalf("exit %d, stderr: %s", code, errs)
			}
			for _, name := range flagNames(t) {
				if !regexp.MustCompile(`--?` + name + `\b|-l ` + name + `\b`).MatchString(out) {
					t.Errorf("flag %q missing from the %s script", name, sh)
				}
			}
			for _, value := range []string{"zsh", "command", "auto", "dracula"} {
				if !strings.Contains(out, value) {
					t.Errorf("value %q missing from the %s script", value, sh)
				}
			}
		})
	}
}

func TestCompletionScriptsParse(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish"} {
		t.Run(sh, func(t *testing.T) {
			bin, err := exec.LookPath(sh)
			if err != nil {
				t.Skipf("%s not installed", sh)
			}
			_, out, _ := run(t, nil, "--completion", sh)
			file := filepath.Join(t.TempDir(), "completion."+sh)
			if err := os.WriteFile(file, []byte(out), 0o600); err != nil {
				t.Fatal(err)
			}
			if msg, err := exec.Command(bin, "-n", file).CombinedOutput(); err != nil {
				t.Fatalf("%s -n: %v\n%s", sh, err, msg)
			}
		})
	}
}

// TestBashCompletion runs the generated function the way bash's completion
// would, with COMP_WORDS split as readline splits them (at "=" too).
func TestBashCompletion(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	_, script, _ := run(t, nil, "--completion", "bash")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zshrc"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		words []string
		want  string
	}{
		{[]string{"--s"}, "--shell --sort"},
		{[]string{"--plain", "--ver"}, "--version"},
		{[]string{"--shell", ""}, "bash zsh fish"},
		{[]string{"--shell", "=", "z"}, "zsh"},
		{[]string{"--sort", "="}, "file name command"},
		{[]string{"-sort", "c"}, "command"},
		{[]string{"--theme", "drac"}, "dracula"},
		{[]string{"--completion", "f"}, "fish"},
		{[]string{"--file", filepath.Join(dir, "z")}, filepath.Join(dir, "zshrc")},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.words, " "), func(t *testing.T) {
			words := append([]string{"pretty-alias"}, tt.words...)
			cmd := exec.Command(bash, "--norc", "--noprofile", "-c", script+`
COMP_WORDS=("$@"); COMP_CWORD=$(($# - 1)); _pretty_alias; echo "${COMPREPLY[*]}"`,
				"bash")
			cmd.Args = append(cmd.Args, words...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if got := strings.TrimSpace(string(out)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompletionUnknownShellIsAUsageError(t *testing.T) {
	code, out, errs := run(t, nil, "--completion", "tcsh")
	if code != ExitUsage || out != "" || !strings.Contains(errs, `unknown shell "tcsh"`) {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
}

func TestShortDesc(t *testing.T) {
	tests := map[string]string{
		"config file to read (default: the shell's usual rc file)": "config file to read",
		"shell dialect to parse: bash, zsh or fish":                "shell dialect to parse",
		"syntax highlighting theme: auto (monokai\nor more)":       "syntax highlighting theme",
		"sort order: file; s cycles it":                            "sort order",
		"print the version and exit":                               "print the version and exit",
	}
	for in, want := range tests {
		if got := shortDesc(in); got != want {
			t.Errorf("shortDesc(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompletionQuoting(t *testing.T) {
	if got := shQuote("it's"); got != `'it'\''s'` {
		t.Errorf("shQuote = %s", got)
	}
	if got := fishQuote(`it's a \`); got != `'it\'s a \\'` {
		t.Errorf("fishQuote = %s", got)
	}
	if got := zshEscape("a [b]: c"); got != `a \[b\]\: c` {
		t.Errorf("zshEscape = %s", got)
	}
}
