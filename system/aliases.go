package system

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

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

	shell, err := GetUserShell()
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

func GetAliases() ([]string, error) {
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
