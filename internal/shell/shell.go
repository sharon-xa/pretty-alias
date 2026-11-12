// Package shell identifies the user's shell and knows where its config lives.
package shell

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Shell is a supported shell.
type Shell string

// Supported shells.
const (
	Bash Shell = "bash"
	Zsh  Shell = "zsh"
	Fish Shell = "fish"
)

// All lists the supported shells.
var All = []Shell{Bash, Zsh, Fish}

// UnsupportedError reports a shell pretty-alias doesn't understand.
type UnsupportedError struct{ Name string }

func (e *UnsupportedError) Error() string {
	if e.Name == "" {
		return "cannot detect your shell ($SHELL is empty); pass --shell bash|zsh|fish"
	}
	return fmt.Sprintf(
		"unsupported shell %q (supported: bash, zsh, fish); pass --shell to override",
		e.Name,
	)
}

// Parse converts a shell name or executable path ("zsh", "/usr/bin/fish")
// into a Shell.
func Parse(nameOrPath string) (Shell, error) {
	// Base name only: "/home/zsh-fan/bin/mysh" must not match "zsh".
	name := strings.ToLower(filepath.Base(strings.TrimSpace(nameOrPath)))
	if name == "." || name == "/" {
		name = ""
	}
	for _, s := range All {
		if name == string(s) {
			return s, nil
		}
	}
	return "", &UnsupportedError{Name: nameOrPath}
}

// Detect works out the shell from the value of $SHELL (the user's login
// shell). Pass os.Getenv("SHELL").
func Detect(shellEnv string) (Shell, error) { return Parse(shellEnv) }

// ConfigFile is the rc file that normally holds the user's aliases.
func (s Shell) ConfigFile(home string) string {
	switch s {
	case Fish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	case Zsh:
		return filepath.Join(home, ".zshrc")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

func (s Shell) String() string { return string(s) }
