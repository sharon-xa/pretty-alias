package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	table        string
	err          error
	width        int
	height       int
	ready        bool
	scrollOffset int
}

func printTable(width int, height int) tea.Cmd {
	return func() tea.Msg {
		aliases, err := getAliases()
		if err != nil {
			return errMsg{err}
		}
		rows := getTableRows(aliases)
		t := getTable(rows, width, height)
		return tableMsg(t.String())
	}
}

type (
	tableMsg string
	errMsg   struct{ err error }
)

func (e errMsg) Error() string { return e.err.Error() }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, printTable(m.width, m.height)

	case tableMsg:
		m.table = string(msg)
		return m, nil

	case errMsg:
		m.err = msg
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down":
			lines := strings.Split(m.table, "\n")
			if len(lines) > m.height {
				maxOffset := len(lines) - m.height
				if m.scrollOffset < maxOffset {
					m.scrollOffset++
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	if !m.ready {
		return "Waiting for window size..."
	}

	tableStr := m.table
	lines := strings.Split(tableStr, "\n")

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
