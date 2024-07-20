package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/lipgloss"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
	cyan      = lipgloss.Color("51")
)

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
