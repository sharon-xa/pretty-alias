package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

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
