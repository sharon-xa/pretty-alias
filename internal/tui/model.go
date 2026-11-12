// Package tui is the Bubble Tea front end: a scrollable, highlighted table of
// aliases with a cursor for copying. It receives already-parsed data and never touches the filesystem
// or the shell, which keeps it trivially testable.
package tui

import (
	"fmt"
	"strings"
	"time"
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

// minTableLines is how much of the table must stay visible for the full help
// to be shown; on shorter windows `?` falls back to the one-line footer.
const minTableLines = 3

// Model is the Bubble Tea model.
type Model struct {
	all    []row  // every alias, in the current order
	rows   []row  // the ones matching the filter (all of them when it is empty)
	nameW  int    // measured over all rows, so filtering doesn't shift the columns
	source string // where the aliases came from, shown in the footer
	order  alias.Order

	// Colors. The background is assumed dark until the terminal says
	// otherwise (see DetectBackground); nothing is drawn before that is known.
	pal        palette
	cursorMark string // the cursor bar, pre-rendered in pal.accent
	detect     bool   // the background is being asked for
	bgKnown    bool   // the answer (or the timeout) has arrived
	lightHL    *highlight.Highlighter

	keys     keyMap
	help     help.Model
	showHelp bool // the full (`?`) help is open
	vp       viewport.Model

	cursor   int    // index into rows
	spans    []span // where each of rows is in the rendered table
	toast    string // "copied …", shown in the footer for a moment
	toastSeq int

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
// The colors are for a dark background; see DetectBackground.
func New(aliases []alias.Alias, hl *highlight.Highlighter, source string) *Model {
	rows := make([]row, len(aliases))
	nameW := 0
	for i, a := range aliases {
		r := row{
			name:    highlight.Sanitize(a.Name),
			command: hl.Command(a.Command),
			text:    highlight.Sanitize(a.Command),
			idx:     i,
		}
		rows[i] = r
		nameW = max(nameW, lipgloss.Width(r.name))
	}

	keys := newKeyMap()
	vp := viewport.New()
	vp.KeyMap = viewport.KeyMap{} // Update handles every key itself
	vp.MouseWheelEnabled = false  // keep the mouse free for selecting/copying text

	in := textinput.New()
	in.Prompt = "/"
	in.Placeholder = "filter by name or command"

	m := &Model{
		all: rows, rows: rows, nameW: nameW, source: highlight.Sanitize(source),
		keys: keys, help: help.New(), vp: vp, filter: in, bgKnown: true,
		order: alias.ByFile,
	}
	m.setPalette(true)
	m.vp.LeftGutterFunc = m.gutter
	return m
}

// Run starts the program on the alternate screen and blocks until it exits.
func Run(m *Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	if !m.detect {
		return nil
	}
	return tea.Batch(
		tea.RequestBackgroundColor,
		tea.Tick(bgTimeout, func(time.Time) tea.Msg { return bgTimeoutMsg{} }),
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case tea.BackgroundColorMsg:
		m.bgKnown = true
		if !msg.IsDark() {
			m.useLight()
		}
		return m, nil

	case bgTimeoutMsg: // no answer: keep the dark colors
		m.bgKnown = true
		return m, nil

	case toastDoneMsg:
		if int(msg) == m.toastSeq {
			m.toast = ""
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.filtering {
			return m, m.updateFilter(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.setHelp(!m.showHelp)
			return m, nil
		case key.Matches(msg, m.keys.CloseHelp): // only enabled while the help is open
			m.setHelp(false)
			return m, nil
		case key.Matches(msg, m.keys.Filter):
			m.setHelp(false) // its keys don't apply while typing
			m.filtering = true
			m.filter.CursorEnd()
			return m, m.filter.Focus()
		case key.Matches(msg, m.keys.ClearFilter):
			m.filter.SetValue("")
			m.applyFilter()
			return m, nil
		case key.Matches(msg, m.keys.Sort):
			m.SetOrder(m.order.Next())
		case key.Matches(msg, m.keys.Copy):
			return m, m.copySelected(false)
		case key.Matches(msg, m.keys.CopyName):
			return m, m.copySelected(true)
		case key.Matches(msg, m.keys.Up):
			m.moveCursor(m.cursor - 1)
		case key.Matches(msg, m.keys.Down):
			m.moveCursor(m.cursor + 1)
		case key.Matches(msg, m.keys.Top):
			m.moveCursor(0)
		case key.Matches(msg, m.keys.Bottom):
			m.moveCursor(len(m.rows) - 1)
		case key.Matches(msg, m.keys.PageDown):
			m.scroll(m.vp.PageDown, true)
		case key.Matches(msg, m.keys.PageUp):
			m.scroll(m.vp.PageUp, false)
		case key.Matches(msg, m.keys.HalfDown):
			m.scroll(m.vp.HalfPageDown, true)
		case key.Matches(msg, m.keys.HalfUp):
			m.scroll(m.vp.HalfPageUp, false)
		}
		return m, nil
	}

	if m.filtering { // cursor blink and other input messages
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		return m, cmd
	}
	return m, nil
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
		m.layout() // the full help can change height with ClearFilter enabled
		m.setContent()
	}
	m.cursor = 0
	m.vp.GotoTop()
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

// resize re-lays the table out for the new size.
func (m *Model) resize(w, h int) {
	m.width, m.height = w, h
	m.help.SetWidth(w)
	m.vp.SetWidth(w)
	m.layout()
	m.setContent()
	m.vp.SetYOffset(m.vp.YOffset()) // clamps into range
	m.showCursor()                  // rows may have changed height
	m.ready = true
}

// layout sizes the viewport and the filter input around the footer. It runs on
// resize and whenever the footer changes height (the `?` help).
func (m *Model) layout() {
	m.vp.SetHeight(max(m.height-m.footerLines(), 1))
	// Clamp the offset: the table may have just lost or gained lines on screen.
	m.vp.SetYOffset(m.vp.YOffset())

	// The input gets whatever the footer line has left, assuming the widest count.
	n := len(m.all)
	countW := len(fmt.Sprintf("  %d/%d aliases  ", n, n))
	helpW := lipgloss.Width(m.help.ShortHelpView(m.keys.filterHelp()))
	promptW := lipgloss.Width(m.filter.Prompt)
	m.filter.SetWidth(max(m.width-1-promptW-1-countW-helpW, 8)) // 1 margin + 1 cursor
}

// setHelp opens or closes the full help and re-lays the screen out.
func (m *Model) setHelp(on bool) {
	m.showHelp = on
	m.keys.CloseHelp.SetEnabled(on)
	if on {
		m.keys.Help.SetHelp("?", "close help")
	} else {
		m.keys.Help.SetHelp("?", "help")
	}
	if m.ready {
		m.layout()
	}
}

func (m *Model) setContent() {
	w := m.width - 1 // the cursor gutter takes one column
	s, spans := renderTable(m.rows, m.nameW, w, m.pal)
	if len(m.rows) == 0 && len(m.all) > 0 {
		msg := lipgloss.NewStyle().Foreground(m.pal.dim).Render("no aliases match the filter")
		s += "\n\n" + lipgloss.PlaceHorizontal(w, lipgloss.Center, msg)
	}
	m.spans = spans
	m.vp.SetContent(s)
}

// View implements tea.Model. The program runs on the alternate screen.
func (m *Model) View() tea.View {
	v := tea.NewView(m.content())
	v.AltScreen = true
	return v
}

func (m *Model) content() string {
	if !m.ready || !m.bgKnown {
		return ""
	}
	if m.footerLines() == 0 {
		return m.vp.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), m.footer())
}

// footerLines is the footer's height: none on windows too short for content
// plus a footer, one line normally, more with the full help open.
func (m *Model) footerLines() int {
	if m.height <= 2 {
		return 0
	}
	if m.fullHelpFits() {
		return lipgloss.Height(m.fullHelp()) + 1
	}
	return 1
}

// fullHelpFits reports whether the `?` help is open and the window is tall
// enough for it next to a few table lines.
func (m *Model) fullHelpFits() bool {
	return m.showHelp && !m.filtering &&
		m.height >= lipgloss.Height(m.fullHelp())+1+minTableLines
}

func (m *Model) fullHelp() string {
	h := m.help // indented by one column, so it gets one column less
	h.SetWidth(max(m.width-1, 1))
	// help drops whole columns that don't fit; MaxWidth is the backstop.
	return lipgloss.NewStyle().PaddingLeft(1).MaxWidth(m.width).
		Render(h.FullHelpView(m.keys.FullHelp()))
}

func (m *Model) footer() string {
	dim := lipgloss.NewStyle().Foreground(m.pal.dim)
	count := fmt.Sprintf("%d aliases", len(m.all))
	if m.query != "" {
		count = fmt.Sprintf("%d/%d aliases", len(m.rows), len(m.all))
	}

	if m.filtering {
		line := " " + m.filter.View() + dim.Render("  "+count+"  ") +
			m.help.ShortHelpView(m.keys.filterHelp())
		return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
	}

	// The keys come first: if the line is too long, the status gives way.
	keys := m.help.ShortHelpView(m.keys.ShortHelp())
	room := m.width - lipgloss.Width(keys)
	if m.toast != "" {
		status := lipgloss.NewStyle().Foreground(m.pal.accent).Render(" ✓ "+m.toast) + "  "
		if room > 0 {
			keys = lipgloss.NewStyle().MaxWidth(room).Render(status) + keys
		}
		return m.withFullHelp(lipgloss.NewStyle().MaxWidth(m.width).Render(keys))
	}
	head := " " + count
	if m.query != "" {
		head += " · /" + highlight.Sanitize(m.query)
	}
	if m.order != alias.ByFile {
		head += " · by " + string(m.order)
	}
	tail := fmt.Sprintf(" · %3.0f%%  ", m.vp.ScrollPercent()*100)
	src := truncateLeft(m.source, room-lipgloss.Width(head)-lipgloss.Width(tail)-len(" · "))
	status := head
	if src != "" {
		status += " · " + src
	}
	status += tail
	if room > 0 { // MaxWidth(0) means no limit
		status = lipgloss.NewStyle().MaxWidth(room).Render(dim.Render(status))
	} else {
		status = ""
	}

	return m.withFullHelp(lipgloss.NewStyle().MaxWidth(m.width).Render(status + keys))
}

// withFullHelp puts the `?` help above the status line when it is open.
func (m *Model) withFullHelp(line string) string {
	if m.fullHelpFits() {
		return lipgloss.JoinVertical(lipgloss.Left, m.fullHelp(), line)
	}
	return line
}

// truncateLeft shortens s to at most w cells by cutting from the left, which
// keeps the file name of a path: "…/fish/config.fish". It returns "" if w is
// too small to show anything useful.
func truncateLeft(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w < 4 {
		return ""
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
		r = r[1:]
	}
	return "…" + string(r)
}
