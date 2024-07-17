package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	aliases := getAliases()
	printAliases(aliases)
}

func getFishConfFile() *string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(
			fmt.Sprintf(
				"ERROR: couldn't get home dir.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}

	fileContent, err := os.ReadFile(home + "/.config/fish/config.fish")
	if err != nil {
		panic(
			fmt.Sprintf(
				"ERROR: couldn't read config file content.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}
	strFileContent := string(fileContent)
	return &strFileContent
}

func getAliases() []string {
	fileContent := getFishConfFile()

	lines := strings.Split(*fileContent, "\n")
	var aliases []string

	for _, line := range lines {
		if len(line) < 10 {
			continue
		}
		if line[:5] == "alias" {
			aliases = append(aliases, line)
		}
	}

	return aliases
}

func printAliases(aliases []string) {
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

	printTable(rows)
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
