package table

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

const (
	Purple    = lipgloss.Color("99")
	Gray      = lipgloss.Color("245")
	LightGray = lipgloss.Color("241")
	Cyan      = lipgloss.Color("51")
)

func highlightCommand(code string) string {
	lexer := lexers.Get("fish")
	if lexer == nil {
		fmt.Println("No lexer found for Bash")
	}

	style := styles.Get("monokai")
	if style == nil {
		fmt.Println("No style found for Monokai")
	}

	formatter := formatters.TTY256

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
