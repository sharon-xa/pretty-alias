package main

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sharon-xa/pretty-alias/system"
	"github.com/sharon-xa/pretty-alias/table"
)

type model struct {
	aliases      []string
	table        string
	err          error
	width        int
	height       int
	ready        bool
	scrollOffset int
}

type (
	aliasesMsg []string
	tableMsg   string
	errMsg     struct{ err error }
)

func (e errMsg) Error() string { return e.err.Error() }

func fetchAliases() tea.Cmd {
	return func() tea.Msg {
		aliases, err := system.GetAliases()
		if err != nil {
			log.Println("couldn't get the aliases\nError:", err)
			return errMsg{err}
		}
		return aliasesMsg(aliases)
	}
}

func renderTable(aliases []string, width int, height int) tea.Cmd {
	return func() tea.Msg {
		rows := table.GetTableRows(aliases)
		log.Println("WIDTH:", width, "HEIGHT:", height)
		t := table.GetTable(rows, width, height)
		return tableMsg(t.String())
	}
}

func (m *model) Init() tea.Cmd {
	if m.ready {
		log.Println("Init called again, but model is already initialized")
		return nil
	}

	log.Println("Init called")
	return tea.Batch(fetchAliases())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case aliasesMsg:
		// If the window size is not yet set, store aliases and wait for WindowSizeMsg
		m.aliases = msg
		if m.width == 0 || m.height == 0 {
			log.Println("Received aliasesMsg but window size not set, waiting...")
			return m, nil
		}
		log.Println("AliasesMsg Update, Width:", m.width, ", Height:", m.height)
		return m, renderTable(m.aliases, m.width, m.height)

	case tableMsg:
		m.table = string(msg)
		return m, nil

	case errMsg:
		m.err = msg
		return m, tea.Quit

	case tea.WindowSizeMsg:
		log.Println("WindowSizeMsg Update, Width:", m.width, ", Height:", m.height)
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, renderTable(m.aliases, m.width, m.height)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
				return m, nil
			}
		case "down":
			lines := strings.Split(m.table, "\n")
			if len(lines) > m.height {
				maxOffset := len(lines) - m.height
				if m.scrollOffset < maxOffset {
					m.scrollOffset++
					return m, nil
				}
			}
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.err != nil {
		log.Println("Error encountered:", m.err)
		return m.err.Error()
	}

	if !m.ready {
		return "Waiting for window size..."
	}

	lines := strings.Split(m.table, "\n")
	start := m.scrollOffset
	end := m.scrollOffset + m.height
	if end > len(lines) {
		end = len(lines)
	}

	display := strings.Join(lines[start:end], "\n")

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		display,
	)
}
