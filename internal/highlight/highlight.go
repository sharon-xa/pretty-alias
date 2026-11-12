// Package highlight turns a command string into terminal-ready, syntax
// highlighted text. It wraps Chroma so the rest of the program never sees it.
//
// Nothing in this package prints, panics or returns an error: if highlighting
// isn't possible the text comes back plain.
package highlight

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/colorprofile"

	"github.com/sharon-xa/pretty-alias/v2/internal/shell"
)

// Highlighter highlights commands for one shell. Create it once and reuse it.
// A nil *Highlighter is valid and returns plain text.
type Highlighter struct {
	lexer     chroma.Lexer
	style     *chroma.Style
	formatter chroma.Formatter
}

// DefaultTheme is the theme used unless one is chosen. On light backgrounds
// AutoTheme picks its light counterpart instead (monokailight).
const DefaultTheme = "monokai"

// Themes returns the names New accepts (every Chroma style), sorted.
func Themes() []string { return styles.Names() }

// IsTheme reports whether name is a known theme. Case doesn't matter.
func IsTheme(name string) bool {
	_, ok := styles.Registry[strings.ToLower(name)]
	return ok
}

// AutoTheme returns DefaultTheme for dark backgrounds and its registered light
// counterpart for light ones.
func AutoTheme(dark bool) string {
	mode := chroma.Light
	if dark {
		mode = chroma.Dark
	}
	return strings.ToLower(styles.GetForMode(DefaultTheme, mode).Name)
}

// New returns a Highlighter using the lexer that matches sh, the Chroma style
// called theme (see IsTheme; unknown names get Chroma's fallback style), and
// the formatter that matches the terminal's colour profile p (see
// colorprofile.Detect, which also honours NO_COLOR). Profiles without colour
// produce plain text.
func New(sh shell.Shell, p colorprofile.Profile, theme string) *Highlighter {
	lexer := lexers.Get(string(sh)) // "bash", "zsh" (-> bash lexer) and "fish" exist
	if lexer == nil {
		lexer = lexers.Fallback
	}

	return &Highlighter{
		lexer:     chroma.Coalesce(lexer),
		style:     styles.Get(theme),
		formatter: chromaFormatter(p),
	}
}

// Command returns s sanitized (see Sanitize) and highlighted.
func (h *Highlighter) Command(s string) string {
	s = Sanitize(s)
	if h == nil || h.lexer == nil || s == "" {
		return s
	}

	it, err := h.lexer.Tokenise(nil, s)
	if err != nil {
		return s
	}

	var b strings.Builder
	if err := h.formatter.Format(&b, h.style, it); err != nil {
		return s
	}

	return strings.TrimRight(b.String(), "\n")
}

// Sanitize makes untrusted text safe to print in a terminal: invalid UTF-8 is
// replaced with U+FFFD and control characters (ESC, BEL, tab, newline, ...) are
// shown as their visible "Control Pictures" symbols (ESC -> ␛) so a config
// file can never move the cursor, change the title or clear the screen.
func Sanitize(s string) string {
	if !strings.ContainsFunc(s, unicode.IsControl) && utf8.ValidString(s) {
		return s
	}

	s = strings.ToValidUTF8(s, "�")
	return strings.Map(func(r rune) rune {
		switch {
		case r < 0x20:
			return 0x2400 + r // U+2400..U+241F
		case r == 0x7f:
			return 0x2421 // ␡
		case unicode.IsControl(r): // C1 controls
			return '�'
		}
		return r
	}, s)
}

func chromaFormatter(p colorprofile.Profile) chroma.Formatter {
	switch p {
	case colorprofile.TrueColor:
		return formatters.TTY16m
	case colorprofile.ANSI256:
		return formatters.TTY256
	case colorprofile.ANSI:
		return formatters.TTY16
	default:
		return formatters.NoOp
	}
}
