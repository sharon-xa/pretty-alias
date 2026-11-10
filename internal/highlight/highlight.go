// Package highlight turns a command string into terminal-ready, syntax
// highlighted text. It wraps Chroma so the rest of the program never sees it.
//
// Nothing in this package prints, panics or returns an error: if highlighting
// isn't possible the text comes back plain.
package highlight

import (
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/sharon-xa/pretty-alias/internal/shell"
)

// Highlighter highlights commands for one shell. Create it once and reuse it.
// A nil *Highlighter is valid and returns plain text.
type Highlighter struct {
	lexer     chroma.Lexer
	style     *chroma.Style
	formatter chroma.Formatter
}

// New returns a Highlighter using the lexer that matches sh. It honours the
// NO_COLOR convention (https://no-color.org) by returning plain output.
func New(sh shell.Shell) *Highlighter {
	if os.Getenv("NO_COLOR") != "" {
		return nil
	}
	lexer := lexers.Get(string(sh)) // "bash", "zsh" (-> bash lexer) and "fish" exist
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	return &Highlighter{lexer: lexer, style: style, formatter: formatters.TTY256}
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
