package tui

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
	"github.com/sharon-xa/pretty-alias/v2/internal/shell"
)

var (
	white = tea.BackgroundColorMsg{Color: color.White}
	black = tea.BackgroundColorMsg{Color: color.Black}
)

const nearWhite = "38;2;248;248;242" // monokai's plain-text color

func detecting(t *testing.T) *Model {
	t.Helper()
	as := []alias.Alias{{Name: "hi", Command: "echo hello world"}, {Name: "x", Command: "ls"}}
	m := New(as, highlight.New(shell.Bash, colorprofile.TrueColor, "monokai"), "x")
	m.DetectBackground(highlight.New(shell.Bash, colorprofile.TrueColor, "monokailight"))
	return m
}

// Regression test for issue #30: on a light background the UI used bright
// cyan (51) and the commands near-white text, both unreadable on white.
func TestLightBackgroundSwitchesColors(t *testing.T) {
	m := detecting(t)
	if m.Init() == nil {
		t.Fatal("Init does not ask for the background color")
	}
	send(m, size(100, 20))
	if v := m.content(); v != "" {
		t.Fatalf("drew a frame before the background was known:\n%s", v)
	}

	send(m, white, press("?"))
	v := m.content()
	if strings.Contains(v, "38;5;51m") || !strings.Contains(v, "38;5;24m") {
		t.Error("light background: UI still uses bright cyan instead of teal 24")
	}
	if strings.Contains(v, nearWhite) {
		t.Error("light background: commands still highlighted with near-white text")
	}
	if !strings.Contains(sgr.ReplaceAllString(v, ""), "echo hello world") {
		t.Errorf("command missing after re-highlighting:\n%s", v)
	}
}

func TestDarkBackgroundKeepsColors(t *testing.T) {
	m := detecting(t)
	send(m, size(100, 20), black)
	v := m.content()
	if !strings.Contains(v, "38;5;51m") || !strings.Contains(v, nearWhite) {
		t.Errorf("dark background: colors changed:\n%q", v)
	}
}

// A terminal that never answers: after the timeout the dark colors are used.
func TestNoBackgroundAnswer(t *testing.T) {
	m := detecting(t)
	send(m, size(100, 20), bgTimeoutMsg{})
	if v := m.content(); v == "" || !strings.Contains(v, "38;5;51m") {
		t.Errorf("after timeout: %q", v)
	}
	// A late light answer still switches.
	send(m, white)
	if strings.Contains(m.content(), "38;5;51m") {
		t.Error("late light answer was ignored")
	}
}

// Switching keeps the filter, the cursor and the scroll position valid.
func TestLightSwitchKeepsState(t *testing.T) {
	m := New(sample(200), highlight.New(shell.Bash, colorprofile.TrueColor, "monokai"), "x")
	m.DetectBackground(highlight.New(shell.Bash, colorprofile.TrueColor, "monokailight"))
	send(m, size(100, 20), bgTimeoutMsg{}, press("/"))
	typeText(m, "a01")
	send(m, press("enter"), press("j"), press("j"), white)
	if len(m.rows) != 100 || m.cursor != 2 || m.filter.Value() != "a01" {
		t.Errorf("rows %d cursor %d query %q", len(m.rows), m.cursor, m.filter.Value())
	}
	assertCursorVisible(t, m)
	if strings.Contains(m.content(), nearWhite) {
		t.Error("filtered rows not re-highlighted")
	}
}

// Without DetectBackground (NO_COLOR, tests): no query, frames right away.
func TestNoDetection(t *testing.T) {
	m := New(sample(3), nil, "x")
	if m.Init() != nil {
		t.Error("Init asks the terminal although detection is off")
	}
	send(m, size(80, 24))
	if m.content() == "" {
		t.Error("no frame without detection")
	}
	send(m, white) // an unrequested answer changes nothing
	if !strings.Contains(m.content(), "38;5;51m") {
		t.Error("colors changed without detection")
	}
}
