package tui

import (
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// toastDuration is how long "copied …" stays in the footer.
const toastDuration = 2 * time.Second

// toastDoneMsg clears the toast with the same sequence number; a newer copy
// has its own timer.
type toastDoneMsg int

var cursorMark = lipgloss.NewStyle().Foreground(colorAccent).Render("▌")

// gutter draws the cursor: a bar left of the selected row's lines. Moving the
// cursor only changes what this returns, so the table is never re-rendered.
func (m *Model) gutter(c viewport.GutterContext) string {
	if m.cursor < len(m.spans) {
		if s := m.spans[m.cursor]; c.Index >= s.first && c.Index <= s.last {
			return cursorMark
		}
	}
	return " "
}

// moveCursor selects row i (clamped) and scrolls just enough to show it. The
// first and last rows also bring the header and bottom border into view.
func (m *Model) moveCursor(i int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor = min(max(i, 0), len(m.rows)-1)
	switch m.cursor {
	case 0:
		m.vp.GotoTop()
	case len(m.rows) - 1:
		m.vp.GotoBottom()
	}
	m.showCursor()
}

// showCursor scrolls the least amount that shows the selected row; if the row
// is taller than the viewport, its first line wins.
func (m *Model) showCursor() {
	if m.cursor >= len(m.spans) {
		return
	}
	s := m.spans[m.cursor]
	top, h := m.vp.YOffset(), m.vp.Height()
	if s.last >= top+h {
		top = s.last - h + 1
	}
	if s.first < top {
		top = s.first
	}
	m.vp.SetYOffset(top)
}

// scroll runs a viewport scroll (page or half page), then pulls the cursor onto
// the screen. If the view can't move any further, the cursor goes to the first
// or last row instead, so paging always gets there.
func (m *Model) scroll(move func(), down bool) {
	before := m.vp.YOffset()
	move()
	if m.vp.YOffset() == before {
		if down {
			m.moveCursor(len(m.rows) - 1)
		} else {
			m.moveCursor(0)
		}
		return
	}
	top, bottom := m.vp.YOffset(), m.vp.YOffset()+m.vp.Height()-1
	if m.cursor >= len(m.spans) {
		return
	}
	if s := m.spans[m.cursor]; s.first < top {
		for m.cursor < len(m.spans)-1 && m.spans[m.cursor].first < top {
			m.cursor++
		}
	} else if s.last > bottom {
		for m.cursor > 0 && m.spans[m.cursor].last > bottom {
			m.cursor--
		}
	}
}

// copySelected puts the selected row's command (or name) on the clipboard via
// OSC 52 and shows a toast. It copies the sanitized text, exactly what is on
// screen: a raw newline from a config file could run a command when pasted.
func (m *Model) copySelected(name bool) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	r := m.rows[m.cursor]
	text, what := r.text, "command"
	if name {
		text, what = r.name, "name"
	}
	m.toastSeq++
	m.toast = "copied " + what + " of " + r.name
	seq := m.toastSeq
	return tea.Batch(
		tea.SetClipboard(text),
		tea.Tick(toastDuration, func(time.Time) tea.Msg { return toastDoneMsg(seq) }),
	)
}
