// Package alias finds and parses alias definitions in shell config files.
//
// It is deliberately UI-free: it takes text in and returns []Alias. The TUI,
// the plain-text output and any future JSON output all consume the same type.
package alias

// Alias is one alias definition found in a config file.
//
// Name and Command are the shell-unquoted values, as the shell itself would
// see them: alias ll='ls -la' gives Name "ll" and Command "ls -la".
type Alias struct {
	Name    string
	Command string
	// Comment is a trailing "# ..." comment on the definition's line, without
	// the leading "#". Empty if there is none.
	Comment string
	// File and Line say where the definition came from (Line is 1-based).
	File string
	Line int
}
