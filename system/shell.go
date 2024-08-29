package system

import (
	"errors"
	"os"
	"strings"
)

// return which shell the user is running.
// only 3 shells are supported: bash, zsh, fish
func GetUserShell() (string, error) {
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
