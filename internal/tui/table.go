package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Colours (ANSI 256 palette).
var (
	colorAccent = lipgloss.Color("51")  // cyan
	colorDim    = lipgloss.Color("245") // gray
)

const (
	margin       = 2  // blank columns each side of the table
	minTableW    = 24 // never lay the table out narrower than this
	minNameW     = 5  // width of the "ALIAS" header
	chromeW      = 7  // 3 border columns + 2×2 cell padding
	maxNameShare = 3  // alias column gets at most 1/maxNameShare of the width
)

// row is one pre-rendered table row.
type row struct {
	name    string // sanitized
	command string // sanitized + highlighted (may contain ANSI codes)
}

// renderTable lays the rows out for a terminal of the given width. It is pure
// (rows + width in, string out), so it is cheap to re-run on every resize.
func renderTable(rows []row, nameW, width int) string {
	tableW := max(width-2*margin, minTableW)

	nameW = min(max(nameW, minNameW), max(tableW/maxNameShare, minNameW))
	cmdW := max(tableW-nameW-chromeW, 8)

	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Align(lipgloss.Center)
	cell := lipgloss.NewStyle().Padding(0, 1)

	data := make([][]string, len(rows))
	for i, r := range rows {
		data[i] = []string{r.name, r.command}
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorAccent)).
		StyleFunc(func(r, col int) lipgloss.Style {
			w := nameW + 2 // lipgloss widths include padding
			if col == 1 {
				w = cmdW + 2
			}
			if r == table.HeaderRow {
				return header.Width(w)
			}
			s := cell.Width(w)
			if r > 0 { // blank line between entries, none under the header
				s = s.PaddingTop(1)
			}
			return s
		}).
		Headers("ALIAS", "COMMAND").
		Rows(data...)

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, t.String())
}
