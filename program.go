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
	rows         [][]string
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

func fetchAliases(m *model) tea.Cmd {
	return func() tea.Msg {
		aliases, err := system.GetAliases()
		if err != nil {
			log.Println("couldn't get the aliases\nError:", err)
			return errMsg{err}
		}
		m.rows = table.GetTableRows(aliases)
		return aliasesMsg(aliases)
	}
}

func renderTable(m *model) tea.Cmd {
	return func() tea.Msg {
		t := m.GetTable()
		return tableMsg(t.String())
	}
}

func (m *model) Init() tea.Cmd {
	if m.ready {
		log.Println("Init called again, but model is already initialized")
		return nil
	}

	log.Println("Init called")
	return fetchAliases(m)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case aliasesMsg:
		// If the window size is not yet set, store aliases and wait for WindowSizeMsg
		if m.width == 0 || m.height == 0 {
			log.Println("Received aliasesMsg but window size not set, waiting...")
			return m, nil
		}
		log.Println("AliasesMsg Update, Width:", m.width, ", Height:", m.height)
		return m, renderTable(m)

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
		return m, renderTable(m)

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
	end := min(m.scrollOffset+m.height, len(lines))

	display := strings.Join(lines[start:end], "\n")

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		display,
	)
}
