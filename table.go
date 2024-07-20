package main

import (
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func cleanAliasCommand(command string) string {
	if len(command) > 1 {
		switch command[0] {
		case '"', '\'':
			if command[len(command)-1] == command[0] {
				command = strings.Trim(command, string(command[0]))
			}
		}
	}
	return command
}

func createFishAliasRows(aliases []string) *[][]string {
	var rows [][]string
	for _, alias := range aliases {
		aliasWithoutTheWord := alias[6:]

		var aliasDeclerationType string

	findingTypeLoop:
		for _, letter := range aliasWithoutTheWord {
			if string(letter) == "=" {
				aliasDeclerationType = "="
				break findingTypeLoop
			} else if string(letter) == " " {
				aliasDeclerationType = " "
				break findingTypeLoop
			}
		}

		var aliasNameAndCommand []string
		if aliasDeclerationType == "=" {
			aliasNameAndCommand = strings.SplitN(aliasWithoutTheWord, "=", 2)
		} else if aliasDeclerationType == " " {
			aliasNameAndCommand = strings.SplitN(aliasWithoutTheWord, " ", 2)
		}

		aliasNameAndCommand[1] = cleanAliasCommand(aliasNameAndCommand[1])

		aliasNameAndCommand[1] = highlightFishCode(aliasNameAndCommand[1])
		rows = append(rows, aliasNameAndCommand)
	}
	return &rows
}

func createBashZshAliasRows(aliases []string) *[][]string {
	var rows [][]string
	for _, alias := range aliases {
		aliasWithoutTheWord := alias[6:]

		var aliasNameAndCommand []string
		aliasNameAndCommand = strings.SplitN(aliasWithoutTheWord, "=", 2)

		aliasNameAndCommand[1] = cleanAliasCommand(aliasNameAndCommand[1])

		aliasNameAndCommand[1] = highlightFishCode(aliasNameAndCommand[1])
		rows = append(rows, aliasNameAndCommand)
	}
	return &rows
}

func getTableRows(aliases []string) *[][]string {
	shell, err := getUserShell()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var rows *[][]string
	if shell == "fish" {
		rows = createFishAliasRows(aliases)
	} else {
		rows = createBashZshAliasRows(aliases)
	}

	return rows
}

func getTable(rows *[][]string, width int, height int) *table.Table {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Foreground(cyan).Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(0, 1)
	)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(cyan)).Height(height).
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
