package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	aliases, err := getAliases()
	if err != nil {
		log.Fatalln(err.Error())
	}
	printAliases(aliases)
}

// return which shell the user is running.
// only 3 shells are supported: bash, zsh, fish
func getUserShell() (string, error) {
	shell := os.Getenv("SHELL")

	if strings.Contains(shell, "fish") {
		return "fish", nil
	} else if strings.Contains(shell, "bash") {
		return "bash", nil
	} else if strings.Contains(shell, "zsh") {
		return "zsh", nil
	}

	return "", errors.New("Shell not found!")
}

func getConfigFile() *string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(
			fmt.Sprintf(
				"ERROR: couldn't get home dir.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}

	shell, err := getUserShell()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var fileContent []byte

	if shell == "fish" {
		fileContent, err = os.ReadFile(home + "/.config/fish/config.fish")
	} else if shell == "bash" {
		fileContent, err = os.ReadFile(home + "/.bashrc")
	} else if shell == "zsh" {
		fileContent, err = os.ReadFile(home + "/.zshrc")
	}

	if err != nil {
		log.Fatalln(
			fmt.Sprintf(
				"ERROR: couldn't read config file content.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}
	strFileContent := string(fileContent)
	return &strFileContent
}

func getAliases() ([]string, error) {
	fileContent := getConfigFile()

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

	if len(aliases) == 0 {
		return aliases, errors.New("no aliases found")
	}

	return aliases, nil
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

func printAliases(aliases []string) {
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
