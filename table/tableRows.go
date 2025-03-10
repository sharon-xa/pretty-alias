package table

import (
	"log"
	"strings"

	"github.com/sharon-xa/pretty-alias/system"
)

func GetTableRows(aliases []string) *[][]string {
	shell, err := system.GetUserShell()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var rows *[][]string
	if shell == "fish" {
		rows = createFishAliasRows(aliases)
	} else {
		rows = createBashZshAliasRows(aliases)
	}

	clear(aliases)
	return rows
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

		aliasNameAndCommand[1] = highlightCommand(aliasNameAndCommand[1])
		rows = append(rows, aliasNameAndCommand)
	}
	clear(aliases)
	return &rows
}

func createBashZshAliasRows(aliases []string) *[][]string {
	var rows [][]string
	for _, alias := range aliases {
		// remove the word alias
		aliasWithoutTheWord := alias[5:]
		TrimedAlias := strings.TrimSpace(aliasWithoutTheWord)

		var aliasNameAndCommand []string
		aliasNameAndCommand = strings.SplitN(TrimedAlias, "=", 2)

		aliasNameAndCommand[1] = cleanAliasCommand(aliasNameAndCommand[1])

		aliasNameAndCommand[1] = highlightCommand(aliasNameAndCommand[1])
		rows = append(rows, aliasNameAndCommand)
	}
	clear(aliases)
	return &rows
}
