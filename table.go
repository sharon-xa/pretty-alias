package main

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	glossTable "github.com/charmbracelet/lipgloss/table"
	"github.com/sharon-xa/pretty-alias/table"
)

func (m *model) GetTable() *glossTable.Table {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Foreground(table.Cyan).Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(0, 1)
	)

	t := glossTable.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(table.Cyan)).Height(m.height).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == glossTable.HeaderRow:
				return HeaderStyle
			// case row%2 == 0: // this is for even row style
			// 	style = CellStyle
			default:
				style = CellStyle
			}

			if row > 1 {
				style = style.PaddingTop(1)
			}

			// second column (commands column)
			if col == 1 {
				style = style.Width(m.width - 20)
			}

			return style
		}).
		Headers("ALIAS", "COMMAND").
		Rows(m.rows...)
	return t
}
