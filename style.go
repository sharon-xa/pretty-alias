package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
	cyan      = lipgloss.Color("51")
)

func printTable(rows *[][]string) {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Foreground(cyan).Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(0, 1)
	)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(cyan)).
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

			// Make the second column a little wider.
			if col == 1 {
				style = style.Width(85)
			}

			return style
		}).
		Headers("ALIAS", "COMMAND").
		Rows(*rows...)
	fmt.Println(t)
}

func highlightFishCode(code string) string {
	lexer := lexers.Get("fish")
	if lexer == nil {
		fmt.Println("No lexer found for Bash")
	}

	style := styles.Get("monokai")
	if style == nil {
		fmt.Println("No style found for Monokai")
	}

	formatter := formatters.TTY8

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		fmt.Println(err)
	}

	var builder strings.Builder
	err = formatter.Format(&builder, style, iterator)
	if err != nil {
		fmt.Println(err)
	}

	return builder.String()
}
