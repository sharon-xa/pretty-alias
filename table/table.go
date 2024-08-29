package table

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func GetTable(rows *[][]string, width int, height int) *table.Table {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Foreground(Cyan).Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(0, 1)
	)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(Cyan)).Height(height).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = CellStyle
			default:
				style = CellStyle
			}

			if row > 1 {
				style = style.PaddingTop(1)
			}

			// second column
			if col == 1 {
				style = style.Width(width - 20)
			}

			return style
		}).
		Headers("ALIAS", "COMMAND").
		Rows(*rows...)
	return t
}
