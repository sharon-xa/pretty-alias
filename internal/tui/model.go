// Package tui is the Bubble Tea front end: a scrollable, highlighted table of
// aliases. It receives already-parsed data and never touches the filesystem
// or the shell, which keeps it trivially testable.
package tui

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
	"github.com/sharon-xa/pretty-alias/v2/internal/highlight"
)

const footerHeight = 1

// Model is the Bubble Tea model.
type Model struct {
	all    []row  // every alias, in file order
	rows   []row  // the ones matching the filter (all of them when it is empty)
	nameW  int    // measured over all rows, so filtering doesn't shift the columns
	source string // where the aliases came from, shown in the footer

	keys keyMap
	help help.Model
	vp   viewport.Model

	// filter holds the query. It has focus while the user is typing; after
	// enter it keeps its value (the filter stays applied) without focus.
	filter    textinput.Model
	filtering bool
	query     string // the query rows were last filtered with

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
		r := row{
			name:    highlight.Sanitize(a.Name),
			command: hl.Command(a.Command),
			text:    highlight.Sanitize(a.Command),
		}
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

	in := textinput.New()
	in.Prompt = "/"
	in.Placeholder = "filter by name or command"
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(colorAccent)
	st.Blurred.Prompt = lipgloss.NewStyle().Foreground(colorAccent)
	in.SetStyles(st)

	return &Model{
		all: rows, rows: rows, nameW: nameW, source: source,
		keys: keys, help: h, vp: vp, filter: in,
	}
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
		if m.filtering {
			return m, m.updateFilter(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Filter):
			m.filtering = true
			m.filter.CursorEnd()
			return m, m.filter.Focus()
		case key.Matches(msg, m.keys.ClearFilter):
			m.filter.SetValue("")
			m.applyFilter()
			return m, nil
		case key.Matches(msg, m.keys.Top):
			m.vp.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.vp.GotoBottom()
			return m, nil
		}
	}

	var cmds []tea.Cmd
	if m.filtering { // cursor blink and other input messages
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		cmds = append(cmds, cmd)
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}

// updateFilter handles a key while the filter input has focus. Printable keys
// (including q, j, k, g…) are text here, not commands.
func (m *Model) updateFilter(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.FilterForceQuit):
		return tea.Quit
	case key.Matches(msg, m.keys.FilterCancel):
		m.filter.SetValue("")
		m.stopFiltering()
		return nil
	case key.Matches(msg, m.keys.FilterAccept):
		m.stopFiltering()
		return nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()
	return cmd
}

func (m *Model) stopFiltering() {
	m.filtering = false
	m.filter.Blur()
	m.applyFilter()
}

// applyFilter recomputes the visible rows if the query changed. It re-renders
// only then, because rendering a large table is the expensive part.
func (m *Model) applyFilter() {
	q := m.filter.Value()
	m.keys.ClearFilter.SetEnabled(q != "")
	if q == m.query {
		return
	}
	m.query = q
	m.rows = filterRows(m.all, q)
	if m.ready {
		m.setContent()
		m.vp.GotoTop()
	}
}

// filterRows keeps the rows whose name or command contains q. Matching is
// smart-case: case-insensitive unless q has an upper-case letter.
func filterRows(rows []row, q string) []row {
	if q == "" {
		return rows
	}
	fold := !strings.ContainsFunc(q, unicode.IsUpper)
	if fold {
		q = strings.ToLower(q)
	}
	var out []row
	for _, r := range rows {
		name, text := r.name, r.text
		if fold {
			name, text = strings.ToLower(name), strings.ToLower(text)
		}
		if strings.Contains(name, q) || strings.Contains(text, q) {
			out = append(out, r)
		}
	}
	return out
}

// resize re-lays the table out for the new size and keeps the scroll position
// valid (the table can get shorter when the window gets wider).
func (m *Model) resize(w, h int) {
	m.width, m.height = w, h
	m.help.SetWidth(w)
	m.vp.SetWidth(w)
	m.filter.SetWidth(max(w/2, 10))
	if m.showFooter() {
		m.vp.SetHeight(h - footerHeight)
	} else {
		m.vp.SetHeight(max(h, 1))
	}
	m.setContent()
	m.vp.SetYOffset(m.vp.YOffset()) // clamps into range
	m.ready = true
}

func (m *Model) setContent() {
	s := renderTable(m.rows, m.nameW, m.width)
	if len(m.rows) == 0 && len(m.all) > 0 {
		msg := lipgloss.NewStyle().Foreground(colorDim).Render("no aliases match the filter")
		s += "\n\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, msg)
	}
	m.vp.SetContent(s)
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
	dim := lipgloss.NewStyle().Foreground(colorDim)
	count := fmt.Sprintf("%d aliases", len(m.all))
	if m.query != "" {
		count = fmt.Sprintf("%d/%d aliases", len(m.rows), len(m.all))
	}

	var line string
	if m.filtering {
		line = " " + m.filter.View() + dim.Render("  "+count+"  ") +
			m.help.ShortHelpView(m.keys.filterHelp())
	} else {
		status := count
		if m.query != "" {
			status += " · /" + highlight.Sanitize(m.query)
		}
		pct := m.vp.ScrollPercent() * 100
		status = fmt.Sprintf(" %s · %s · %3.0f%%  ", status, m.source, pct)
		line = dim.Render(status) + m.help.ShortHelpView(m.keys.ShortHelp())
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}
