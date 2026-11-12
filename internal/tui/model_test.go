package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
	"github.com/sharon-xa/pretty-alias/v2/internal/shell"
)

func sample(n int) []alias.Alias {
	out := make([]alias.Alias, n)
	for i := range out {
		out[i] = alias.Alias{
			Name: fmt.Sprintf("a%04d", i),
			Command: fmt.Sprintf(
				"echo %d && some-long-command --with --many --flags --so-it-wraps-when-narrow --index=%d",
				i,
				i,
			),
		}
	}
	return out
}

func newModel(t *testing.T, n int) *Model {
	t.Helper()
	return New(
		sample(n),
		highlight.New(shell.Bash, colorprofile.ANSI256, highlight.DefaultTheme),
		"~/.bashrc",
	)
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
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+u":
		return tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	case "ctrl+d":
		return tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func TestNothingRenderedBeforeSizeKnown(t *testing.T) {
	m := newModel(t, 3)
	if v := m.content(); v != "" {
		t.Errorf("View before WindowSizeMsg = %q, want empty", v)
	}
}

func TestViewShowsHeaderRowsAndFooter(t *testing.T) {
	m := newModel(t, 5)
	send(m, size(100, 30))
	v := m.content()
	wants := []string{"ALIAS", "COMMAND", "a0000", "a0004", "5 aliases", "~/.bashrc", "quit"}
	for _, want := range wants {
		if !strings.Contains(v, want) {
			t.Errorf("View missing %q:\n%s", want, v)
		}
	}
}

func TestViewFitsTheWindow(t *testing.T) {
	for _, sz := range [][2]int{{120, 40}, {80, 24}, {40, 20}, {20, 10}, {10, 5}, {1, 1}} {
		m := newModel(t, 200)
		send(m, size(sz[0], sz[1]))
		v := m.content()
		if got := lipgloss.Height(v); got > sz[1] {
			t.Errorf("%dx%d: view is %d lines tall", sz[0], sz[1], got)
		}
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
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

// j/k move the cursor one alias at a time and the view follows; paging moves
// the view and pulls the cursor along.
func TestCursorMovement(t *testing.T) {
	m := newModel(t, 300)
	send(m, size(100, 20))
	if m.cursor != 0 || m.vp.YOffset() != 0 {
		t.Fatalf("starts at cursor %d, offset %d", m.cursor, m.vp.YOffset())
	}
	send(m, press("down"), press("down"), press("j"))
	if m.cursor != 3 {
		t.Errorf("after 3 downs: cursor %d, want 3", m.cursor)
	}
	send(m, press("up"), press("k"))
	if m.cursor != 1 {
		t.Errorf("after 2 ups: cursor %d, want 1", m.cursor)
	}
	assertCursorVisible(t, m)

	for range 40 {
		send(m, press("j"))
		assertCursorVisible(t, m)
	}
	if m.vp.YOffset() == 0 {
		t.Error("view did not follow the cursor down")
	}

	send(m, press("pgdown"))
	assertCursorVisible(t, m)
	send(m, press("G"))
	if m.cursor != 299 || !m.vp.AtBottom() {
		t.Errorf("G: cursor %d, at bottom %v", m.cursor, m.vp.AtBottom())
	}
	send(m, press("down"), press("down")) // must not go past the end
	if m.cursor != 299 || !m.vp.AtBottom() {
		t.Error("moved past the last row")
	}
	send(m, press("g"))
	if m.cursor != 0 || !m.vp.AtTop() {
		t.Errorf("g: cursor %d, at top %v", m.cursor, m.vp.AtTop())
	}
	send(m, press("up"))
	if m.cursor != 0 || m.vp.YOffset() != 0 {
		t.Errorf("moved above the top: cursor %d, offset %d", m.cursor, m.vp.YOffset())
	}
	send(m, press("space")) // v2 reports the space bar as "space", not " "
	if m.vp.YOffset() == 0 || m.cursor == 0 {
		t.Errorf("space did not page down: offset %d, cursor %d", m.vp.YOffset(), m.cursor)
	}
	assertCursorVisible(t, m)
}

// Paging always reaches the ends, even when the view can't scroll any further.
func TestPagingReachesFirstAndLastRow(t *testing.T) {
	m := newModel(t, 50)
	send(m, size(100, 20))
	for range 100 {
		send(m, press("pgdown"))
		assertCursorVisible(t, m)
	}
	if m.cursor != 49 {
		t.Errorf("pgdown x100: cursor %d, want 49", m.cursor)
	}
	for range 100 {
		send(m, press("ctrl+u"))
		assertCursorVisible(t, m)
	}
	if m.cursor != 0 {
		t.Errorf("ctrl+u x100: cursor %d, want 0", m.cursor)
	}
}

// assertCursorVisible checks that the selected row's first line is on screen
// and carries the cursor mark.
func assertCursorVisible(t *testing.T, m *Model) {
	t.Helper()
	s := m.spans[m.cursor]
	top := m.vp.YOffset()
	if s.first < top || s.first >= top+m.vp.Height() {
		t.Fatalf("cursor row %d (lines %d-%d) is off screen (offset %d, height %d)",
			m.cursor, s.first, s.last, top, m.vp.Height())
	}
	line := strings.Split(m.vp.View(), "\n")[s.first-top]
	if !strings.HasPrefix(sgr.ReplaceAllString(line, ""), "▌") {
		t.Fatalf("cursor row's first line has no mark: %q", line)
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
		t.Fatalf(
			"test premise broken: table did not shrink (%d -> %d)",
			tall,
			m.vp.TotalLineCount(),
		)
	}
	if max := m.vp.TotalLineCount() - m.vp.Height(); m.vp.YOffset() > max {
		t.Errorf("offset %d beyond max %d after resize", m.vp.YOffset(), max)
	}
	_ = m.content() // must not panic

	// and back again, and a bunch of random sizes
	for _, s := range [][2]int{{30, 10}, {300, 60}, {5, 3}, {1, 1}, {0, 0}, {80, 24}} {
		send(m, size(s[0], s[1]), press("G"), press("g"), press("pgdown"))
		_ = m.content()
	}
}

func TestTinyAndZeroSizesDoNotPanic(t *testing.T) {
	for _, s := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {5, 1}, {19, 3}} {
		m := newModel(t, 10)
		send(m, size(s[0], s[1]))
		_ = m.content()
	}
}

func TestNoAliases(t *testing.T) {
	m := newModel(t, 0)
	send(m, size(80, 24), press("down"), press("G"))
	_ = m.content() // header only, no panic
}

func TestHostileContentIsNeutralised(t *testing.T) {
	as := []alias.Alias{{Name: "bad\x1b[2Jname", Command: "echo \x1b]0;title\x07 \x1b[31m"}}
	for _, p := range []colorprofile.Profile{colorprofile.Ascii, colorprofile.TrueColor} {
		m := New(as, highlight.New(shell.Bash, p, highlight.DefaultTheme), "x")
		send(m, size(100, 20))
		v := m.content()
		if strings.Contains(v, "\x1b[2J") || strings.Contains(v, "\x1b]0;") ||
			strings.Contains(v, "\x07") {
			t.Errorf("%s: raw escape sequence reached the screen: %q", p, v)
		}
	}
}

func TestUnicodeFixtureLikeRows(t *testing.T) {
	as := []alias.Alias{
		{Name: "日本語", Command: "echo こんにちは世界"},
		{Name: "🚀", Command: "echo \"🚀✨🔥\""},
		{Name: strings.Repeat("n", 200), Command: strings.Repeat("x", 3000)},
	}
	m := New(as, highlight.New(shell.Zsh, colorprofile.ANSI256, highlight.DefaultTheme), "x")
	for _, w := range []int{120, 60, 30, 12} {
		send(m, size(w, 30))
		v := m.content()
		for i, line := range strings.Split(v, "\n") {
			if lw := lipgloss.Width(line); lw > w && w >= minTableW {
				t.Errorf("width %d: line %d is %d cells wide", w, i, lw)
				break
			}
		}
	}
}

func BenchmarkResize2000(b *testing.B) {
	m := New(sample(2000), nil, "x")
	for i := 0; i < b.N; i++ {
		m.Update(size(100+i%50, 40))
	}
}

func BenchmarkNew2000(b *testing.B) {
	as := sample(2000)
	hl := highlight.New(shell.Bash, colorprofile.ANSI256, highlight.DefaultTheme)
	for i := 0; i < b.N; i++ {
		New(as, hl, "x")
	}
}

// typeText sends s one key at a time, as a user typing it would.
func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// esc clears a filter; it never quits (q and ctrl+c do).
func TestEscDoesNotQuit(t *testing.T) {
	m := newModel(t, 3)
	send(m, size(80, 24))
	if _, cmd := m.Update(press("esc")); isQuit(cmd) {
		t.Error("esc quit")
	}
}

func TestFilterNarrowsRows(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(120, 40), press("/"))
	typeText(m, "a001")
	if len(m.rows) != 10 {
		t.Fatalf("a001 matched %d rows, want 10", len(m.rows))
	}
	v := m.content()
	for _, want := range []string{"a0010", "a0019", "10/200 aliases"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "a0020") {
		t.Errorf("view shows a non-matching row:\n%s", v)
	}
}

func TestFilterMatchesCommand(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(120, 40), press("/"))
	typeText(m, "--index=42")
	if len(m.rows) != 1 || m.rows[0].name != "a0042" {
		t.Errorf("--index=42 matched %v, want only a0042", m.rows)
	}
}

func TestFilterIsSmartCase(t *testing.T) {
	as := []alias.Alias{{Name: "gst", Command: "git status"}, {Name: "GS", Command: "Git Stash"}}
	tests := []struct {
		query string
		want  int
	}{
		{"gs", 2},  // all lower case: case-insensitive
		{"git", 2}, // matches "git status" and "Git Stash"
		{"Git", 1}, // has an upper-case letter: exact case
		{"GS", 1},
	}
	for _, tt := range tests {
		m := New(as, nil, "x")
		send(m, size(80, 24), press("/"))
		typeText(m, tt.query)
		if len(m.rows) != tt.want {
			t.Errorf("%q matched %d rows, want %d", tt.query, len(m.rows), tt.want)
		}
	}
}

// While typing, letters that are normally commands (q quits, j/g scroll) are
// part of the query.
func TestFilterTakesCommandLetters(t *testing.T) {
	m := newModel(t, 3)
	send(m, size(80, 24), press("/"))
	for _, k := range []string{"q", "j", "g", "G", "f", "/"} {
		if _, cmd := m.Update(press(k)); isQuit(cmd) {
			t.Fatalf("%q quit while filtering", k)
		}
	}
	if got := m.filter.Value(); got != "qjgGf/" {
		t.Errorf("query = %q, want %q", got, "qjgGf/")
	}
	if _, cmd := m.Update(press("ctrl+c")); !isQuit(cmd) {
		t.Error("ctrl+c did not quit while filtering")
	}
}

func TestFilterEnterKeepsEscClears(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 20), press("/"))
	typeText(m, "a00")
	send(m, press("enter"))
	if m.filtering || len(m.rows) != 100 {
		t.Fatalf("after enter: filtering=%v rows=%d, want false/100", m.filtering, len(m.rows))
	}
	send(m, press("j")) // keys are commands again
	if m.cursor != 1 || m.filter.Value() != "a00" {
		t.Fatalf("j after enter: cursor %d, query %q", m.cursor, m.filter.Value())
	}
	if v := m.content(); !strings.Contains(v, "100/200 aliases · /a00") {
		t.Errorf("footer does not show the active filter:\n%s", v)
	}

	// "/" again edits the same query.
	send(m, press("/"), press("backspace"))
	if m.filter.Value() != "a0" {
		t.Errorf("re-opened query = %q, want a0", m.filter.Value())
	}
	send(m, press("enter"))

	// esc with a filter applied clears it, and doesn't quit.
	if _, cmd := m.Update(press("esc")); isQuit(cmd) {
		t.Fatal("esc quit with a filter applied")
	}
	if len(m.rows) != 200 || m.filter.Value() != "" {
		t.Errorf("after esc: rows=%d query=%q", len(m.rows), m.filter.Value())
	}
	if v := m.content(); !strings.Contains(v, "200 aliases") || strings.Contains(v, "/200") {
		t.Errorf("footer after clearing:\n%s", v)
	}
}

func TestFilterEscWhileTypingClears(t *testing.T) {
	m := newModel(t, 50)
	send(m, size(100, 20), press("/"))
	typeText(m, "a004")
	send(m, press("esc"))
	if m.filtering || m.filter.Value() != "" || len(m.rows) != 50 {
		t.Errorf("filtering=%v query=%q rows=%d", m.filtering, m.filter.Value(), len(m.rows))
	}
}

func TestFilterScrollsToTop(t *testing.T) {
	m := newModel(t, 300)
	send(m, size(100, 20), press("G"), press("/"))
	typeText(m, "a01")
	if !m.vp.AtTop() {
		t.Errorf("offset %d after filtering, want top", m.vp.YOffset())
	}
}

func TestFilterNoMatches(t *testing.T) {
	m := newModel(t, 20)
	send(m, size(100, 20), press("/"))
	typeText(m, "zzz")
	v := m.content()
	if !strings.Contains(v, "no aliases match") || !strings.Contains(v, "0/20 aliases") {
		t.Errorf("no-match view:\n%s", v)
	}
	send(m, press("down"), press("G"), size(30, 5), size(0, 0), size(80, 24))
	_ = m.content()
}

func TestFilterSurvivesResize(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 20), press("/"))
	typeText(m, "a01")
	send(m, press("enter"), size(40, 10), size(200, 50))
	if len(m.rows) != 100 {
		t.Errorf("rows after resize = %d, want 100", len(m.rows))
	}
	if v := m.content(); strings.Contains(v, "a0200") || !strings.Contains(v, "a0100") {
		t.Errorf("view after resize:\n%s", v)
	}
}

func TestFilteringViewFitsTheWindow(t *testing.T) {
	for _, sz := range [][2]int{{120, 40}, {80, 24}, {20, 10}, {10, 5}, {1, 1}} {
		m := newModel(t, 200)
		send(m, size(sz[0], sz[1]), press("/"))
		typeText(m, strings.Repeat("a0", 40))
		v := m.content()
		if got := lipgloss.Height(v); got > sz[1] {
			t.Errorf("%dx%d: view is %d lines tall", sz[0], sz[1], got)
		}
		if sz[0] >= minTableW {
			for _, line := range strings.Split(v, "\n") {
				if lw := lipgloss.Width(line); lw > sz[0] {
					t.Errorf("%dx%d: line is %d cells wide", sz[0], sz[1], lw)
					break
				}
			}
		}
	}
}

func TestFilterQueryIsSanitized(t *testing.T) {
	m := newModel(t, 5)
	send(m, size(100, 20), press("/"))
	m.Update(tea.PasteMsg{Content: "a\x1b[2J\x1b]0;x\x07"})
	send(m, press("enter"))
	if v := m.content(); strings.Contains(v, "\x1b[2J") || strings.Contains(v, "\x1b]0;") {
		t.Errorf("raw escape sequence from the query reached the screen: %q", v)
	}
}

func BenchmarkFilterKeystroke2000(b *testing.B) {
	m := New(sample(2000), nil, "x")
	m.Update(size(120, 40))
	m.Update(press("/"))
	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			typeText(m, "a1")
		} else {
			m.Update(press("backspace"))
			m.Update(press("backspace"))
		}
	}
}

var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain returns the view without color codes, for matching text that is
// styled in several pieces (such as "? help").
func plain(m *Model) string { return sgr.ReplaceAllString(m.content(), "") }

// lastLine returns the bottom line of the view (the footer's status line),
// without color codes.
func lastLine(m *Model) string {
	lines := strings.Split(plain(m), "\n")
	return lines[len(lines)-1]
}

// Regression test for issue #27: the footer was wider than 80 columns, so
// "q quit" (the one thing it most needs to say) was cut off. It must fit in 60.
func TestFooterKeepsQuitVisible(t *testing.T) {
	long := "~/" + strings.Repeat("very-long-directory/", 10) + "config.fish"
	for _, w := range []int{120, 80, 60} {
		for _, filtered := range []bool{false, true} {
			m := New(sample(157), nil, long)
			send(m, size(w, 24))
			if filtered {
				send(m, press("/"))
				typeText(m, "a00")
				send(m, press("enter"))
			}
			line := lastLine(m)
			if lw := lipgloss.Width(line); lw > w {
				t.Errorf("w=%d filtered=%v: footer is %d cells wide", w, filtered, lw)
			}
			for _, want := range []string{"q quit", "? help", "/ filter", "y copy"} {
				if !strings.Contains(line, want) {
					t.Errorf("w=%d filtered=%v: footer lacks %q: %q", w, filtered, want, line)
				}
			}
			if filtered && !strings.Contains(line, "esc clear filter") {
				t.Errorf("w=%d: footer lacks the clear-filter key: %q", w, line)
			}
		}
	}
}

func TestFooterShortensLongSourcePath(t *testing.T) {
	long := "~/" + strings.Repeat("dir/", 20) + "config.fish"
	m := New(sample(5), nil, long)
	send(m, size(80, 24))
	line := lastLine(m)
	if !strings.Contains(line, "…") || !strings.Contains(line, "config.fish") {
		t.Errorf("long path not shortened to its tail: %q", line)
	}
	send(m, size(200, 24))
	if line := lastLine(m); !strings.Contains(line, long) {
		t.Errorf("path shortened although it fits: %q", line)
	}
}

func TestHelpToggle(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 30))
	short := m.vp.Height()
	if strings.Contains(plain(m), "half page down") {
		t.Fatal("full help shown before ?")
	}

	send(m, press("?"))
	v := plain(m)
	for _, want := range []string{"half page down", "bottom", "page up", "? close help"} {
		if !strings.Contains(v, want) {
			t.Errorf("full help lacks %q:\n%s", want, v)
		}
	}
	if m.vp.Height() >= short {
		t.Errorf("table did not make room for the help: %d -> %d", short, m.vp.Height())
	}
	if got := lipgloss.Height(v); got != 30 {
		t.Errorf("view is %d lines tall, want 30", got)
	}

	send(m, press("?"))
	if strings.Contains(plain(m), "half page down") || m.vp.Height() != short {
		t.Errorf("? did not close the help (viewport %d, want %d)", m.vp.Height(), short)
	}
}

// esc closes the help first, then clears the filter, and never quits.
func TestEscClosesHelpBeforeClearingFilter(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 30), press("/"))
	typeText(m, "a01")
	send(m, press("enter"), press("?"))
	if !m.showHelp {
		t.Fatal("? did not open the help")
	}

	if _, cmd := m.Update(press("esc")); isQuit(cmd) {
		t.Fatal("esc quit")
	}
	if m.showHelp || m.filter.Value() != "a01" {
		t.Fatalf("first esc: help=%v query=%q, want closed help, filter kept",
			m.showHelp, m.filter.Value())
	}
	send(m, press("esc"))
	if m.filter.Value() != "" {
		t.Errorf("second esc did not clear the filter: %q", m.filter.Value())
	}
	if _, cmd := m.Update(press("esc")); isQuit(cmd) {
		t.Error("esc quit")
	}
}

func TestQuestionMarkIsTextWhileFiltering(t *testing.T) {
	m := newModel(t, 20)
	send(m, size(100, 30), press("?"), press("/")) // opening the filter closes the help
	if m.showHelp {
		t.Error("help still open while typing a filter")
	}
	typeText(m, "a?")
	if m.showHelp || m.filter.Value() != "a?" {
		t.Errorf("help=%v query=%q, want ? typed as text", m.showHelp, m.filter.Value())
	}
}

// Opening the help shrinks the table viewport; at the bottom of the table the
// offset must be clamped, the same failure mode as issue #5.
func TestHelpWhileScrolledToBottom(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 30), press("G"), press("?"))
	if max := m.vp.TotalLineCount() - m.vp.Height(); m.vp.YOffset() > max {
		t.Errorf("offset %d beyond max %d", m.vp.YOffset(), max)
	}
	send(m, press("?"))
	if max := m.vp.TotalLineCount() - m.vp.Height(); m.vp.YOffset() > max {
		t.Errorf("after closing: offset %d beyond max %d", m.vp.YOffset(), max)
	}
	_ = m.content()
}

func TestHelpFitsTheWindow(t *testing.T) {
	for _, sz := range [][2]int{{120, 40}, {80, 24}, {60, 10}, {40, 8}, {20, 6}, {10, 3}, {1, 1}} {
		m := newModel(t, 200)
		send(m, size(sz[0], sz[1]), press("?"))
		v := plain(m)
		if got := lipgloss.Height(v); got > sz[1] {
			t.Errorf("%dx%d: view is %d lines tall", sz[0], sz[1], got)
		}
		if sz[0] >= minTableW {
			for _, line := range strings.Split(v, "\n") {
				if lw := lipgloss.Width(line); lw > sz[0] {
					t.Errorf("%dx%d: line is %d cells wide: %q", sz[0], sz[1], lw, line)
					break
				}
			}
		}
		// toggling and resizing must never panic
		send(m, size(sz[1], sz[0]), press("?"), press("G"), press("?"), size(sz[0], sz[1]))
		_ = m.content()
	}
}

// Too short for the full help: ? falls back to the one-line footer.
func TestHelpFallsBackOnShortWindows(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 6), press("?"))
	if strings.Contains(plain(m), "half page down") {
		t.Errorf("full help squeezed into a 6-line window:\n%s", plain(m))
	}
	if !strings.Contains(lastLine(m), "? close help") {
		t.Errorf("footer does not say the help is open: %q", lastLine(m))
	}
	send(m, size(100, 30))
	if !strings.Contains(plain(m), "half page down") {
		t.Error("full help not shown after the window grew")
	}
}

func TestSourceLabelIsSanitized(t *testing.T) {
	m := New(sample(3), nil, "~/evil\x1b]0;x\x07\x1b[2J.zshrc")
	send(m, size(120, 20))
	if v := m.content(); strings.Contains(v, "\x1b[2J") || strings.Contains(v, "\x1b]0;") {
		t.Errorf("raw escape sequence from the source label reached the screen: %q", v)
	}
}
