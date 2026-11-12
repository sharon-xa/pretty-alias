package tui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
)

// bgTimeout is how long the first frame waits for the terminal to report its
// background color. Terminals answer within milliseconds; ones that never
// answer get the dark colors after this.
const bgTimeout = 150 * time.Millisecond

type bgTimeoutMsg struct{}

// DetectBackground makes the program ask the terminal for its background
// color when it starts. If the answer is light, the UI switches to its light
// palette and the commands are re-highlighted with light (nil means plain).
//
// The answer arrives through Bubble Tea's normal input, so nothing blocks and
// no keystrokes are lost; the first frame waits for it for at most bgTimeout.
func (m *Model) DetectBackground(light *highlight.Highlighter) {
	m.detect, m.bgKnown, m.lightHL = true, false, light
}

// useLight switches to the light palette and highlighter, once.
func (m *Model) useLight() {
	if !m.detect {
		return
	}
	m.detect = false // a second answer changes nothing
	m.setPalette(false)
	for i := range m.all {
		m.all[i].command = m.lightHL.Command(m.all[i].text)
	}
	m.rows = filterRows(m.all, m.query)
	if m.ready {
		m.setContent()
		m.showCursor()
	}
}

// setPalette styles the UI (help, filter prompt, cursor) for a dark or light
// background. The table picks up m.pal on its next render.
func (m *Model) setPalette(dark bool) {
	m.pal = newPalette(dark)
	accent := lipgloss.NewStyle().Foreground(m.pal.accent)
	dim := lipgloss.NewStyle().Foreground(m.pal.dim)

	m.help.Styles.ShortKey, m.help.Styles.FullKey = accent, accent
	m.help.Styles.ShortDesc, m.help.Styles.FullDesc = dim, dim
	m.help.Styles.ShortSeparator, m.help.Styles.FullSeparator = dim, dim

	st := textinput.DefaultDarkStyles()
	if !dark {
		st = textinput.DefaultLightStyles()
	}
	st.Focused.Prompt, st.Blurred.Prompt = accent, accent
	m.filter.SetStyles(st)

	m.cursorMark = accent.Render("▌")
}
