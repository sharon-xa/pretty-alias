// Package tui is the Bubble Tea front end: a scrollable, highlighted table of
// aliases. It receives already-parsed data and never touches the filesystem
// or the shell, which keeps it trivially testable.
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
)

const footerHeight = 1

// Model is the Bubble Tea model.
type Model struct {
	rows   []row
	nameW  int
	source string // where the aliases came from, shown in the footer

	keys keyMap
	help help.Model
	vp   viewport.Model

	width, height int
	ready         bool
}

// New prepares the model. All the slow work (sanitizing and highlighting every
// command) happens here, once, before the terminal is taken over.
// hl may be nil for plain output. source is a display label such as "~/.zshrc".
func New(aliases []alias.Alias, hl *highlight.Highlighter, source string) *Model {
	rows := make([]row, len(aliases))
	nameW := 0
	for i, a := range aliases {
		r := row{name: highlight.Sanitize(a.Name), command: hl.Command(a.Command)}
		rows[i] = r
		nameW = max(nameW, lipgloss.Width(r.name))
	}

	keys := newKeyMap()
	vp := viewport.New()
	vp.KeyMap = keys.viewportKeys()
	vp.MouseWheelEnabled = false // keep the mouse free for selecting/copying text

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(colorAccent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(colorDim)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(colorDim)

	return &Model{rows: rows, nameW: nameW, source: source, keys: keys, help: h, vp: vp}
}

// Run starts the program on the alternate screen and blocks until it exits.
func Run(m *Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Top):
			m.vp.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.vp.GotoBottom()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// resize re-lays the table out for the new size and keeps the scroll position
// valid (the table can get shorter when the window gets wider).
func (m *Model) resize(w, h int) {
	m.width, m.height = w, h
	m.help.SetWidth(w)
	m.vp.SetWidth(w)
	if m.showFooter() {
		m.vp.SetHeight(h - footerHeight)
	} else {
		m.vp.SetHeight(max(h, 1))
	}
	m.vp.SetContent(renderTable(m.rows, m.nameW, w))
	m.vp.SetYOffset(m.vp.YOffset()) // clamps into range
	m.ready = true
}

// View implements tea.Model. The program runs on the alternate screen.
func (m *Model) View() tea.View {
	v := tea.NewView(m.content())
	v.AltScreen = true
	return v
}

func (m *Model) content() string {
	if !m.ready {
		return ""
	}
	if !m.showFooter() {
		return m.vp.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), m.footer())
}

// showFooter is false on windows too short to fit content plus a footer.
func (m *Model) showFooter() bool { return m.height > footerHeight+1 }

func (m *Model) footer() string {
	pct := m.vp.ScrollPercent() * 100
	status := lipgloss.NewStyle().Foreground(colorDim).
		Render(fmt.Sprintf(" %d aliases · %s · %3.0f%%  ", len(m.rows), m.source, pct))
	line := status + m.help.ShortHelpView(m.keys.ShortHelp())
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}
