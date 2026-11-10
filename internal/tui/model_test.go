package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sharon-xa/pretty-alias/internal/alias"
	"github.com/sharon-xa/pretty-alias/internal/highlight"
	"github.com/sharon-xa/pretty-alias/internal/shell"
)

func sample(n int) []alias.Alias {
	out := make([]alias.Alias, n)
	for i := range out {
		out[i] = alias.Alias{
			Name:    fmt.Sprintf("a%04d", i),
			Command: fmt.Sprintf("echo %d && some-long-command --with --many --flags --so-it-wraps-when-narrow --index=%d", i, i),
		}
	}
	return out
}

func newModel(t *testing.T, n int) *Model {
	t.Helper()
	t.Setenv("NO_COLOR", "")
	return New(sample(n), highlight.New(shell.Bash), "~/.bashrc")
}

func send(m *Model, msgs ...tea.Msg) {
	for _, msg := range msgs {
		m.Update(msg)
	}
}

func size(w, h int) tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }
func press(s string) tea.Msg {
	switch s {
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestNothingRenderedBeforeSizeKnown(t *testing.T) {
	m := newModel(t, 3)
	if v := m.View(); v != "" {
		t.Errorf("View before WindowSizeMsg = %q, want empty", v)
	}
}

func TestViewShowsHeaderRowsAndFooter(t *testing.T) {
	m := newModel(t, 5)
	send(m, size(100, 30))
	v := m.View()
	for _, want := range []string{"ALIAS", "COMMAND", "a0000", "a0004", "5 aliases", "~/.bashrc", "quit"} {
		if !strings.Contains(v, want) {
			t.Errorf("View missing %q:\n%s", want, v)
		}
	}
}

func TestViewFitsTheWindow(t *testing.T) {
	for _, sz := range [][2]int{{120, 40}, {80, 24}, {40, 20}, {20, 10}, {10, 5}, {1, 1}} {
		m := newModel(t, 200)
		send(m, size(sz[0], sz[1]))
		v := m.View()
		if got := lipgloss.Height(v); got > sz[1] {
			t.Errorf("%dx%d: view is %d lines tall", sz[0], sz[1], got)
		}
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		m := newModel(t, 3)
		send(m, size(80, 24))
		_, cmd := m.Update(press(k))
		if cmd == nil {
			t.Fatalf("%q: no command returned", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%q did not quit", k)
		}
	}
}

func TestScrolling(t *testing.T) {
	m := newModel(t, 300)
	send(m, size(100, 20))
	if m.vp.YOffset != 0 {
		t.Fatalf("starts at %d", m.vp.YOffset)
	}
	send(m, press("down"), press("down"), press("j"))
	if m.vp.YOffset != 3 {
		t.Errorf("after 3 downs: offset %d, want 3", m.vp.YOffset)
	}
	send(m, press("up"), press("k"))
	if m.vp.YOffset != 1 {
		t.Errorf("after 2 ups: offset %d, want 1", m.vp.YOffset)
	}
	send(m, press("pgdown"))
	if m.vp.YOffset <= 1 {
		t.Errorf("pgdown did not move: %d", m.vp.YOffset)
	}
	send(m, press("G"))
	if !m.vp.AtBottom() {
		t.Error("G did not reach the bottom")
	}
	send(m, press("down"), press("down")) // must not go past the end
	if !m.vp.AtBottom() {
		t.Error("scrolled past the bottom")
	}
	send(m, press("g"))
	if !m.vp.AtTop() {
		t.Error("g did not reach the top")
	}
	send(m, press("up"))
	if m.vp.YOffset != 0 {
		t.Errorf("scrolled above the top: %d", m.vp.YOffset)
	}
}

// Regression test for issue #5: scrolling to the bottom of a tall (narrow)
// table and then making the window wider shrinks the table; the old code kept
// a stale offset and View() panicked with "slice bounds out of range".
func TestResizeWhileScrolledDoesNotPanic(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(40, 20))
	send(m, press("G"))
	tall := m.vp.TotalLineCount()
	send(m, size(200, 20))
	if m.vp.TotalLineCount() >= tall {
		t.Fatalf("test premise broken: table did not shrink (%d -> %d)", tall, m.vp.TotalLineCount())
	}
	if max := m.vp.TotalLineCount() - m.vp.Height; m.vp.YOffset > max {
		t.Errorf("offset %d beyond max %d after resize", m.vp.YOffset, max)
	}
	_ = m.View() // must not panic

	// and back again, and a bunch of random sizes
	for _, s := range [][2]int{{30, 10}, {300, 60}, {5, 3}, {1, 1}, {0, 0}, {80, 24}} {
		send(m, size(s[0], s[1]), press("G"), press("g"), press("pgdown"))
		_ = m.View()
	}
}

func TestTinyAndZeroSizesDoNotPanic(t *testing.T) {
	for _, s := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {5, 1}, {19, 3}} {
		m := newModel(t, 10)
		send(m, size(s[0], s[1]))
		_ = m.View()
	}
}

func TestNoAliases(t *testing.T) {
	m := newModel(t, 0)
	send(m, size(80, 24), press("down"), press("G"))
	_ = m.View() // header only, no panic
}

func TestHostileContentIsNeutralised(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	as := []alias.Alias{{Name: "bad\x1b[2Jname", Command: "echo \x1b]0;title\x07 \x1b[31m"}}
	m := New(as, highlight.New(shell.Bash), "x")
	send(m, size(100, 20))
	v := m.View()
	if strings.Contains(v, "\x1b[2J") || strings.Contains(v, "\x1b]0;") || strings.Contains(v, "\x07") {
		t.Errorf("raw escape sequence reached the screen: %q", v)
	}
}

func TestUnicodeFixtureLikeRows(t *testing.T) {
	as := []alias.Alias{
		{Name: "日本語", Command: "echo こんにちは世界"},
		{Name: "🚀", Command: "echo \"🚀✨🔥\""},
		{Name: strings.Repeat("n", 200), Command: strings.Repeat("x", 3000)},
	}
	m := New(as, highlight.New(shell.Zsh), "x")
	for _, w := range []int{120, 60, 30, 12} {
		send(m, size(w, 30))
		v := m.View()
		for i, line := range strings.Split(v, "\n") {
			if lw := lipgloss.Width(line); lw > w && w >= minTableW {
				t.Errorf("width %d: line %d is %d cells wide", w, i, lw)
				break
			}
		}
	}
}

func BenchmarkResize2000(b *testing.B) {
	b.Setenv("NO_COLOR", "1")
	m := New(sample(2000), nil, "x")
	for i := 0; i < b.N; i++ {
		m.Update(size(100+i%50, 40))
	}
}

func BenchmarkNew2000(b *testing.B) {
	as := sample(2000)
	hl := highlight.New(shell.Bash)
	for i := 0; i < b.N; i++ {
		New(as, hl, "x")
	}
}
