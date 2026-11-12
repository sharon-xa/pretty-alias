package tui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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
	text    string // sanitized, unhighlighted command, for filtering
}

// span is the lines a row's text occupies in the rendered table (inclusive,
// 0-based, without the blank line above it).
type span struct{ first, last int }

// headerLines is the top border, the header row and the line under it.
const headerLines = 3

// renderTable lays the rows out for a terminal of the given width and returns
// it with each row's span, which the cursor uses. It is pure (rows + width in),
// so it is cheap to re-run on every resize.
func renderTable(rows []row, nameW, width int) (string, []span) {
	tableW := max(width-2*margin, minTableW)

	nameW = min(max(nameW, minNameW), max(tableW/maxNameShare, minNameW))
	cmdW := max(tableW-nameW-chromeW, 8)

	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Align(lipgloss.Center)
	cell := lipgloss.NewStyle().Padding(0, 1)

	// A row is as tall as its tallest cell; the cells wrap exactly as the table
	// will wrap them because they're rendered with the same styles.
	nameCell, cmdCell := cell.Width(nameW+2), cell.Width(cmdW+2)
	data := make([][]string, len(rows))
	spans := make([]span, len(rows))
	line := headerLines
	for i, r := range rows {
		data[i] = []string{r.name, r.command}
		if i > 0 {
			line++ // the blank line between entries
		}
		h := 1 // most rows fit on one line; only render the ones that may not
		if lipgloss.Width(r.name) > nameW || lipgloss.Width(r.text) > cmdW {
			h = max(
				lipgloss.Height(nameCell.Render(r.name)),
				lipgloss.Height(cmdCell.Render(r.command)),
			)
		}
		spans[i] = span{line, line + h - 1}
		line += h
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

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, t.String()), spans
}
