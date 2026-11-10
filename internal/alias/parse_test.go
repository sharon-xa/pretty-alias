package alias

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/sharon-xa/pretty-alias/internal/shell"
)

// pairs renders aliases as "name=command" for compact comparisons.
func pairs(as []Alias) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Name + "=" + a.Command
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		sh   shell.Shell
		src  string
		want []string
	}{
		// --- basics
		{"double quotes", shell.Bash, `alias ll="ls -la"`, []string{"ll=ls -la"}},
		{"single quotes", shell.Bash, `alias gs='git status'`, []string{"gs=git status"}},
		{"bare word", shell.Bash, `alias l=ls`, []string{"l=ls"}},
		{"empty value", shell.Bash, `alias e=''`, []string{"e="}},
		{"equals inside value", shell.Bash, `alias a='FOO=bar env'`, []string{"a=FOO=bar env"}},
		{"not an alias", shell.Bash, "export A=1\nalias_x=1\naliasing foo=bar", nil},
		{"commented out", shell.Bash, `# alias ll='ls'`, nil},
		{"empty input", shell.Bash, ``, nil},

		// --- issue #3: short lines starting with 'a'
		{"short a lines", shell.Bash, "a\nab\nabc\nabcd\nalia\nalias x=y", []string{"x=y"}},
		{"bare alias word", shell.Bash, "alias", nil},

		// --- issue #4: no value
		{"bash no value", shell.Bash, "alias foo\nalias -p\nalias bar baz\nalias ok=1", []string{"ok=1"}},
		{"zsh no value", shell.Zsh, "alias foo\nalias -L\nalias -m 'g*'\nalias ok=1", []string{"ok=1"}},
		{"fish no value", shell.Fish, "alias\nalias x\nalias onlyname\nalias ok 'ls'", []string{"ok=ls"}},
		{"spaces around equals", shell.Bash, `alias a = 'b'`, nil},

		// --- issue #8: several per line
		{
			"multiple per line", shell.Bash, `alias a="ls" b="pwd" c='date'`,
			[]string{"a=ls", "b=pwd", "c=date"},
		},
		{"separator ends command", shell.Bash, `alias a=b; echo hi`, []string{"a=b"}},
		{"and ends command", shell.Bash, `alias a='x' && echo hi`, []string{"a=x"}},

		// --- issue #9: indentation
		{"spaces indent", shell.Bash, "  alias a=1", []string{"a=1"}},
		{"tab indent", shell.Bash, "\t\talias a=1", []string{"a=1"}},
		{"fish indent", shell.Fish, "    alias a 'x'", []string{"a=x"}},

		// --- issue #10: flags
		{"zsh global", shell.Zsh, `alias -g L='| less'`, []string{"L=| less"}},
		{"zsh suffix", shell.Zsh, `alias -s txt=nvim`, []string{"txt=nvim"}},
		{"double dash", shell.Bash, `alias -- -='cd -'`, []string{"-=cd -"}},
		{"double dash name", shell.Bash, `alias -- --help='echo x'`, []string{"--help=echo x"}},
		{"fish save", shell.Fish, `alias --save ll 'ls -la'`, []string{"ll=ls -la"}},
		{"fish -s", shell.Fish, `alias -s gs 'git status'`, []string{"gs=git status"}},
		{"fish description", shell.Fish, `alias --description 'list files' lst 'ls -l'`, []string{"lst=ls -l"}},

		// --- issue #11: quotes and escapes
		{"dq inside sq", shell.Bash, `alias a='echo "hi there"'`, []string{`a=echo "hi there"`}},
		{"sq inside dq", shell.Bash, `alias a="echo 'hi there'"`, []string{`a=echo 'hi there'`}},
		{"quoted quote", shell.Bash, `alias a='"hi"'`, []string{`a="hi"`}},
		{"escaped dq", shell.Bash, `alias a="echo \"x\""`, []string{`a=echo "x"`}},
		{"escaped dollar", shell.Bash, `alias a="echo \$HOME"`, []string{`a=echo $HOME`}},
		{"dollar in sq", shell.Bash, `alias a='echo $HOME'`, []string{`a=echo $HOME`}},
		{"sq escape idiom", shell.Bash, `alias a='echo it'\''s'`, []string{`a=echo it's`}},
		{"backslash in sq", shell.Bash, `alias a='echo a\\b'`, []string{`a=echo a\\b`}},
		{"backslash in dq stays", shell.Bash, `alias a="echo a\nb"`, []string{`a=echo a\nb`}},
		{"ansi-c quoting", shell.Bash, `alias a=$'x\ty'`, []string{"a=x\ty"}},
		{"quoted name", shell.Zsh, `alias 'open'='xdg-open'`, []string{"open=xdg-open"}},
		{"escaped space in name", shell.Bash, `alias ls\ space='x'`, []string{"ls space=x"}},
		{"unterminated dq", shell.Bash, `alias a="echo mismatch'`, nil},
		{"unterminated sq", shell.Bash, `alias a='oops`, nil},
		{"lone quote", shell.Bash, `alias a="`, nil},
		{"trailing space kept", shell.Bash, `alias a='ls '`, []string{"a=ls "}},
		{"fish sq escape", shell.Fish, `alias a 'it\'s'`, []string{"a=it's"}},
		{"bash sq no escape", shell.Bash, `alias a='x\'`, []string{`a=x\`}},

		// --- issue #12: comments
		{"trailing comment", shell.Bash, `alias ll='ls -la' # list files`, []string{"ll=ls -la"}},
		{"hash in quotes", shell.Bash, `alias a='echo "a # b"' # real comment`, []string{`a=echo "a # b"`}},
		{"hash in unquoted word", shell.Bash, `alias a=b#c`, []string{"a=b#c"}},
		{"comment after bare", shell.Bash, `alias u=ls # note`, []string{"u=ls"}},

		// --- fish forms
		{"fish space", shell.Fish, `alias ll 'ls -la'`, []string{"ll=ls -la"}},
		{"fish equals", shell.Fish, `alias ll='ls -la'`, []string{"ll=ls -la"}},
		{"fish bare words joined", shell.Fish, `alias a5 echo unquoted words`, []string{"a5=echo unquoted words"}},
		{"fish env prefix", shell.Fish, `alias a7 FOO=bar env`, []string{"a7=FOO=bar env"}},
		{"fish equals inside quoted", shell.Fish, `alias a10 'echo a=b'`, []string{"a10=echo a=b"}},
		{"fish empty value", shell.Fish, `alias a17 ''`, []string{"a17="}},
		{"fish equals empty", shell.Fish, `alias a18=""`, []string{"a18="}},
		{"fish name with equals rejected", shell.Fish, `alias gs=git status`, nil},

		// --- file-level quirks
		{"crlf", shell.Bash, "alias a=\"x\"\r\nalias b='y'\r\n", []string{"a=x", "b=y"}},
		{"bom", shell.Bash, utf8BOM + "alias a=1\nalias b=2", []string{"a=1", "b=2"}},
		{"no trailing newline", shell.Bash, "alias a=1\nalias b=2", []string{"a=1", "b=2"}},
		{"duplicates kept", shell.Bash, "alias a=1\nalias a=2", []string{"a=1", "a=2"}},
		{"invalid utf8 survives", shell.Bash, "alias a=\"echo \xc3\x28\"", []string{"a=echo \xc3\x28"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pairs(Parse(tt.sh, tt.src, "test"))
			if !equal(got, tt.want) {
				t.Errorf("Parse(%q)\n got: %q\nwant: %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestParseRecordsPosition(t *testing.T) {
	src := "# header\n\nalias a=1 b=2 # note\n  alias c=3\n"
	got := Parse(shell.Bash, src, "rc")
	want := []Alias{
		{Name: "a", Command: "1", Comment: "note", File: "rc", Line: 3},
		{Name: "b", Command: "2", Comment: "note", File: "rc", Line: 3},
		{Name: "c", Command: "3", File: "rc", Line: 4},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d aliases, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("alias %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestParseNeverPanics feeds garbage and truncations of real lines.
func TestParseNeverPanics(t *testing.T) {
	seeds := []string{
		`alias a='b' c="d" e=f`, `alias -- -='x'`, `alias $'a\`, `alias "`, `alias '`, `alias \`,
		"alias a=\x00\x01\x02", "alias \xff\xfe=1", `alias ===`, `alias =x`, `alias -`, `alias --`,
		`alias a=$'`, `alias a=$'\`, `alias  # only comment`, `alias ;`, `alias & |`,
	}
	for _, sh := range shell.All {
		for _, s := range seeds {
			for n := 0; n <= len(s); n++ {
				Parse(sh, s[:n], "x") // must not panic
			}
		}
	}
}

// --- fixtures ---------------------------------------------------------------

const testdata = "../../testdata"

func load(t *testing.T, sh shell.Shell, rel string) []Alias {
	t.Helper()
	as, err := Load(sh, filepath.Join(testdata, rel))
	if err != nil {
		t.Fatal(err)
	}
	return as
}

// The hand-written regular fixtures must parse every alias line they contain.
func TestRegularFixturesParseEveryAliasLine(t *testing.T) {
	re := regexp.MustCompile(`^alias\s`)
	for _, f := range []struct {
		rel string
		sh  shell.Shell
	}{
		{"bashrc", shell.Bash}, {"zshrc", shell.Zsh}, {"config.fish", shell.Fish},
	} {
		t.Run(f.rel, func(t *testing.T) {
			want := countLines(t, filepath.Join(testdata, f.rel), re)
			got := load(t, f.sh, f.rel)
			if len(got) != want {
				t.Errorf("parsed %d aliases, file has %d alias lines", len(got), want)
			}
			for _, a := range got {
				if a.Name == "" || strings.ContainsAny(a.Name, "'\"") {
					t.Errorf("line %d: bad name %q", a.Line, a.Name)
				}
			}
		})
	}
}

// The generated large fixtures declare how many aliases they contain.
func TestLargeFixtures(t *testing.T) {
	for _, f := range []struct {
		rel string
		sh  shell.Shell
	}{
		{"large/bashrc", shell.Bash}, {"large/zshrc", shell.Zsh}, {"large/config.fish", shell.Fish},
	} {
		t.Run(f.rel, func(t *testing.T) {
			want := headerCount(t, filepath.Join(testdata, f.rel))
			got := load(t, f.sh, f.rel)
			if len(got) != want {
				t.Fatalf("parsed %d aliases, header says %d", len(got), want)
			}
			seen := map[string]bool{}
			for _, a := range got {
				if seen[a.Name] {
					t.Fatalf("duplicate name %q", a.Name)
				}
				seen[a.Name] = true
			}
		})
	}
}

// Expected row counts for the edge fixtures (see testdata/README.md).
func TestEdgeFixtures(t *testing.T) {
	tests := []struct {
		rel  string
		sh   shell.Shell
		want int
	}{
		{"edge/short-a-lines.bashrc", shell.Bash, 2},
		{"edge/alias-without-value.bashrc", shell.Bash, 2},
		{"edge/alias-without-value.zshrc", shell.Zsh, 1},
		{"edge/alias-without-value.config.fish", shell.Fish, 1},
		{"edge/multiple-per-line.bashrc", shell.Bash, 6},
		{"edge/indented.bashrc", shell.Bash, 7},
		{"edge/indented.config.fish", shell.Fish, 4},
		{"edge/flags.zshrc", shell.Zsh, 11},
		{"edge/flags.config.fish", shell.Fish, 4},
		{"edge/quotes-escapes.bashrc", shell.Bash, 17}, // 20 lines, 3 intentionally invalid
		{"edge/trailing-comments.bashrc", shell.Bash, 7},
		{"edge/duplicates.bashrc", shell.Bash, 7},
		{"edge/special-names.bashrc", shell.Bash, 25},
		{"edge/fish-separators.config.fish", shell.Fish, 18},
		{"edge/crlf.bashrc", shell.Bash, 4},
		{"edge/bom.bashrc", shell.Bash, 2},
		{"edge/no-trailing-newline.bashrc", shell.Bash, 2},
		{"edge/unicode-wide.bashrc", shell.Bash, 16},
		{"edge/very-long-command.bashrc", shell.Bash, 6},
		{"edge/ansi-escapes.bashrc", shell.Bash, 4},
		{"edge/bad-bytes.bashrc", shell.Bash, 4},
		{"minimal/bashrc", shell.Bash, 1},
		{"minimal/zshrc", shell.Zsh, 1},
		{"minimal/config.fish", shell.Fish, 1},
		{"empty/bashrc", shell.Bash, 0},
		{"empty/zshrc", shell.Zsh, 0},
		{"empty/config.fish", shell.Fish, 0},
		{"empty/zero-bytes.bashrc", shell.Bash, 0},
		{"empty/blank-lines.bashrc", shell.Bash, 0},
		// entry points only; following `source` is a future feature
		{"sourced/bashrc", shell.Bash, 1},
		{"sourced/zshrc", shell.Zsh, 1},
		{"sourced/config.fish", shell.Fish, 1},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			if got := load(t, tt.sh, tt.rel); len(got) != tt.want {
				t.Errorf("got %d aliases, want %d: %q", len(got), tt.want, pairs(got))
			}
		})
	}
}

func TestSpecificFixtureContents(t *testing.T) {
	t.Run("multiple per line", func(t *testing.T) {
		got := pairs(load(t, shell.Bash, "edge/multiple-per-line.bashrc"))
		want := []string{"a=ls", "b=pwd", "c=date", "d=whoami", "e=hostname", "f=echo hi"}
		if !equal(got, want) {
			t.Errorf("got %q", got)
		}
	})
	t.Run("flags are not names", func(t *testing.T) {
		for _, a := range load(t, shell.Zsh, "edge/flags.zshrc") {
			if strings.HasPrefix(a.Name, "-g") || strings.HasPrefix(a.Name, "-s") || strings.Contains(a.Name, " ") {
				t.Errorf("flag leaked into name: %q", a.Name)
			}
		}
	})
	t.Run("comments are not commands", func(t *testing.T) {
		for _, a := range load(t, shell.Bash, "edge/trailing-comments.bashrc") {
			if strings.Contains(a.Command, "# ") && a.Name != "hashtag" {
				t.Errorf("%s: comment leaked into command %q", a.Name, a.Command)
			}
		}
	})
	t.Run("bom keeps first alias", func(t *testing.T) {
		got := load(t, shell.Bash, "edge/bom.bashrc")
		if len(got) == 0 || got[0].Name != "ll" {
			t.Errorf("first alias lost: %q", pairs(got))
		}
	})
	t.Run("crlf has no carriage returns", func(t *testing.T) {
		for _, a := range load(t, shell.Bash, "edge/crlf.bashrc") {
			if strings.Contains(a.Command, "\r") || strings.Contains(a.Name, "\r") {
				t.Errorf("CR leaked: %+v", a)
			}
		}
	})
}

// --- helpers -----------------------------------------------------------------

func countLines(t *testing.T, path string, re *regexp.Regexp) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if re.MatchString(sc.Text()) {
			n++
		}
	}
	return n
}

var headerRe = regexp.MustCompile(`aliases=(\d+)`)

func headerCount(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := strings.Cut(string(b), "\n")
	m := headerRe.FindStringSubmatch(first)
	if m == nil {
		t.Fatalf("%s: no pa-fixture header in %q", path, first)
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func BenchmarkParseLarge(b *testing.B) {
	data, err := os.ReadFile(filepath.Join(testdata, "large/zshrc"))
	if err != nil {
		b.Skip(err)
	}
	src := string(data)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if n := len(Parse(shell.Zsh, src, "x")); n != 2000 {
			b.Fatal(fmt.Sprint("parsed ", n))
		}
	}
}
