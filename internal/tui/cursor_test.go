package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
	"github.com/sharon-xa/pretty-alias/v2/internal/shell"
)

// The spans renderTable computes must match where lipgloss actually puts each
// row, including wrapped names and commands and wide characters.
func TestSpansMatchRenderedRows(t *testing.T) {
	as := []alias.Alias{
		{Name: "short", Command: "ls"},
		{Name: strings.Repeat("longname", 6), Command: "echo a"},
		{Name: "wrap", Command: strings.Repeat("word ", 60)},
		{Name: "日本語", Command: "echo こんにちは世界 " + strings.Repeat("字", 50)},
		{Name: "🚀", Command: `echo "🚀✨🔥"`},
		{Name: "last", Command: "true"},
	}
	for _, w := range []int{200, 120, 80, 40, 24} {
		m := New(as, highlight.New(shell.Bash, colorprofile.ANSI256), "x")
		send(m, size(w, 200))
		lines := strings.Split(sgr.ReplaceAllString(m.vp.GetContent(), ""), "\n")
		for i, s := range m.spans {
			if s.last >= len(lines) {
				t.Fatalf("w=%d row %d: span %v beyond %d lines", w, i, s, len(lines))
			}
			// The row's first line holds the start of its name...
			first := []rune(m.rows[i].name)[0]
			if !strings.ContainsRune(lines[s.first], first) {
				t.Errorf("w=%d row %d: first line %q lacks %q", w, i, lines[s.first], first)
			}
			// ...the line above it is the blank separator (or the header rule)...
			above := strings.Trim(lines[s.first-1], " │")
			if above != "" && !strings.HasPrefix(above, "├") {
				t.Errorf("w=%d row %d: line above is not blank: %q", w, i, lines[s.first-1])
			}
			// ...and the line below it is the next separator or the bottom border.
			below := strings.Trim(lines[s.last+1], " │")
			if below != "" && !strings.HasPrefix(below, "└") {
				t.Errorf("w=%d row %d: row continues past its span: %q", w, i, lines[s.last+1])
			}
		}
	}
}

func TestCursorMarksOnlySelectedRow(t *testing.T) {
	as := []alias.Alias{
		{Name: "a", Command: "one"},
		{Name: "b", Command: strings.Repeat("two ", 40)}, // wraps at 60 columns
		{Name: "c", Command: "three"},
	}
	m := New(as, nil, "x")
	send(m, size(60, 30), press("j"))
	var marked []int
	for i, line := range strings.Split(sgr.ReplaceAllString(m.vp.View(), ""), "\n") {
		if strings.HasPrefix(line, "▌") {
			marked = append(marked, i)
		}
	}
	s := m.spans[1]
	if s.last == s.first {
		t.Fatal("test premise broken: row b does not wrap")
	}
	if len(marked) != s.last-s.first+1 || marked[0] != s.first {
		t.Errorf("marked lines %v, want %d-%d", marked, s.first, s.last)
	}
}

// A row taller than the screen: its first line is what gets shown.
func TestCursorOnRowTallerThanScreen(t *testing.T) {
	as := []alias.Alias{
		{Name: "a", Command: "x"},
		{Name: "huge", Command: strings.Repeat("y ", 2000)},
	}
	m := New(as, nil, "x")
	send(m, size(40, 10), press("j"))
	if m.vp.YOffset() > m.spans[1].first {
		t.Errorf("offset %d hides the start of the row at %d", m.vp.YOffset(), m.spans[1].first)
	}
	send(m, press("pgdown"), press("pgdown"), press("k"), press("j"))
	_ = m.content()
}

func TestFilterResetsCursor(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(100, 20), press("G"), press("/"))
	typeText(m, "a01")
	if m.cursor != 0 || !m.vp.AtTop() {
		t.Errorf("cursor %d, at top %v after filtering", m.cursor, m.vp.AtTop())
	}
	send(m, press("enter"), press("G"))
	if m.cursor != len(m.rows)-1 {
		t.Errorf("G in filtered rows: cursor %d, want %d", m.cursor, len(m.rows)-1)
	}
}

func TestResizeKeepsCursorVisible(t *testing.T) {
	m := newModel(t, 200)
	send(m, size(200, 30))
	for range 50 {
		send(m, press("j"))
	}
	for _, s := range [][2]int{{40, 30}, {300, 15}, {80, 24}, {30, 8}} {
		send(m, size(s[0], s[1]))
		assertCursorVisible(t, m)
		if m.cursor != 50 {
			t.Errorf("resize to %v moved the cursor to %d", s, m.cursor)
		}
	}
}

// clipboard runs cmd (and any batch inside it) and returns what it would put
// on the clipboard, or "" if nothing.
func clipboard(t *testing.T, cmd tea.Cmd) string {
	t.Helper()
	if cmd == nil {
		return ""
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			if c == nil {
				continue
			}
			if got := clipboardOf(c); got != "" {
				return got
			}
		}
	default:
		return clipboardOf(func() tea.Msg { return msg })
	}
	return ""
}

// clipboardOf runs c and reports the text if it sets the clipboard. The toast
// timer (tea.Tick) would block for its whole duration, so it is given up on.
func clipboardOf(c tea.Cmd) string {
	done := make(chan tea.Msg, 1)
	go func() { done <- c() }()
	select {
	case msg := <-done:
		if fmt.Sprintf("%T", msg) == "tea.setClipboardMsg" {
			return fmt.Sprint(msg)
		}
	case <-time.After(100 * time.Millisecond): // the toast's tea.Tick
	}
	return ""
}

func TestCopyKeys(t *testing.T) {
	as := []alias.Alias{
		{Name: "ll", Command: "ls -la"},
		{Name: "gs", Command: "git status"},
	}
	m := New(as, highlight.New(shell.Bash, colorprofile.TrueColor), "x")
	send(m, size(80, 24), press("j"))

	_, cmd := m.Update(press("y"))
	if got := clipboard(t, cmd); got != "git status" {
		t.Errorf("y copied %q, want the plain command %q", got, "git status")
	}
	if line := lastLine(m); !strings.Contains(line, "copied command of gs") ||
		!strings.Contains(line, "q quit") {
		t.Errorf("footer after y: %q", line)
	}

	_, cmd = m.Update(press("Y"))
	if got := clipboard(t, cmd); got != "gs" {
		t.Errorf("Y copied %q, want %q", got, "gs")
	}
	_, cmd = m.Update(press("enter"))
	if got := clipboard(t, cmd); got != "git status" {
		t.Errorf("enter copied %q", got)
	}
}

// A toast is cleared by its own timer, not by an older one.
func TestToastClears(t *testing.T) {
	m := newModel(t, 3)
	send(m, size(80, 24), press("y"))
	first := m.toastSeq
	send(m, press("Y"))
	send(m, toastDoneMsg(first))
	if m.toast == "" {
		t.Error("the first copy's timer cleared the second copy's toast")
	}
	send(m, toastDoneMsg(m.toastSeq))
	if m.toast != "" || !strings.Contains(lastLine(m), "3 aliases") {
		t.Errorf("toast not cleared: %q", lastLine(m))
	}
}

// Untrusted commands are copied as shown: a newline from the config file must
// not reach the clipboard, where pasting it into a shell would run a command.
func TestCopyIsSanitized(t *testing.T) {
	as := []alias.Alias{{Name: "x", Command: "echo hi\nrm -rf ~\x1b[2J"}}
	m := New(as, nil, "x")
	send(m, size(80, 24))
	_, cmd := m.Update(press("y"))
	got := clipboard(t, cmd)
	if got == "" || strings.ContainsAny(got, "\n\x1b") {
		t.Errorf("copied %q, want control characters neutralised", got)
	}
}

func TestCopyWithNothingSelected(t *testing.T) {
	m := newModel(t, 5)
	send(m, size(80, 24), press("/"))
	typeText(m, "zzz")
	send(m, press("enter"))
	if _, cmd := m.Update(press("y")); cmd != nil {
		t.Error("y with no matching rows returned a command")
	}
	m = newModel(t, 0)
	send(m, size(80, 24))
	if _, cmd := m.Update(press("y")); cmd != nil {
		t.Error("y with no aliases returned a command")
	}
	send(m, press("j"), press("G"), press("pgdown"), press("k"))
	_ = m.content()
}
