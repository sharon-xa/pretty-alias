package highlight

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/colorprofile"
	"github.com/sharon-xa/pretty-alias/internal/shell"
)

func TestSanitize(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "ls -la", "ls -la"},
		{"unicode untouched", "echo 日本語 🚀 café", "echo 日本語 🚀 café"},
		{"escape", "echo \x1b[31mred\x1b[0m", "echo ␛[31mred␛[0m"},
		{"bell", "a\x07b", "a␇b"},
		{"tab and newline", "a\tb\nc", "a␉b␊c"},
		{"delete", "a\x7fb", "a␡b"},
		{"c1 control", "a\u0085b", "a�b"},
		{"invalid utf8", "echo \xc3\x28", "echo �("},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeOutputIsAlwaysSafe(t *testing.T) {
	var all strings.Builder
	for r := rune(0); r < 0x100; r++ {
		all.WriteRune(r)
	}
	all.WriteString("\xff\xfe\xc3")
	got := Sanitize(all.String())
	if !utf8.ValidString(got) {
		t.Error("output is not valid UTF-8")
	}
	if strings.ContainsFunc(got, unicode.IsControl) {
		t.Errorf("control character survived: %q", got)
	}
}

func TestCommand(t *testing.T) {
	const cmd = "git log --oneline | head -n 5 && echo \"done\""
	tests := []struct {
		profile colorprofile.Profile
		want    string // substring proving the right formatter ran; "" = plain
	}{
		{colorprofile.TrueColor, "\x1b[38;2;"},
		{colorprofile.ANSI256, "\x1b[38;5;"},
		{colorprofile.ANSI, "\x1b["},
		{colorprofile.Ascii, ""},
		{colorprofile.NoTTY, ""},
	}
	for _, sh := range shell.All {
		for _, tt := range tests {
			out := New(sh, tt.profile).Command(cmd)
			if tt.want == "" {
				if out != cmd {
					t.Errorf("%s/%s: got %q, want plain %q", sh, tt.profile, out, cmd)
				}
				continue
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("%s/%s: expected %q in %q", sh, tt.profile, tt.want, out)
			}
			if tt.profile == colorprofile.ANSI && strings.Contains(out, "38;") {
				t.Errorf(
					"%s/%s: 256/true-colour codes in 16-colour output: %q",
					sh,
					tt.profile,
					out,
				)
			}
			if strings.HasSuffix(out, "\n") {
				t.Errorf("%s/%s: trailing newline in %q", sh, tt.profile, out)
			}
		}
	}
}

func TestCommandNeverEmitsRawControlsFromInput(t *testing.T) {
	out := New(shell.Bash, colorprofile.TrueColor).Command("echo \x1b]0;pwned\x07 \x1b[2J")
	// The only escapes allowed are the highlighter's own SGR colour codes.
	stripped := out
	for {
		i := strings.Index(stripped, "\x1b[")
		if i < 0 {
			break
		}
		j := strings.IndexByte(stripped[i:], 'm')
		if j < 0 {
			t.Fatalf("non-SGR escape in %q", out)
		}
		stripped = stripped[:i] + stripped[i+j+1:]
	}
	if strings.ContainsAny(stripped, "\x1b\x07") {
		t.Errorf("raw control character from input survived: %q", out)
	}
}

// NO_COLOR (https://no-color.org) is honoured through colour-profile
// detection: it must yield a profile that highlights to plain text.
func TestNoColor(t *testing.T) {
	p := colorprofile.Env([]string{"NO_COLOR=1", "TERM=xterm-256color", "COLORTERM=truecolor"})
	if got := New(shell.Zsh, p).Command("ls -la"); got != "ls -la" {
		t.Errorf("NO_COLOR (profile %s) output = %q, want plain", p, got)
	}
}

func TestNilHighlighterIsPlain(t *testing.T) {
	var h *Highlighter
	if got := h.Command("a\x1bb"); got != "a␛b" {
		t.Errorf("got %q", got)
	}
}
