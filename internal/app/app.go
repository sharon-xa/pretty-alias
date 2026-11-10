// Package app is the composition root: it parses flags, wires the other
// packages together (shell → alias → highlight → tui) and decides exit codes.
// It is the only package that knows about all the others.
package app

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/sharon-xa/pretty-alias/internal/alias"
	"github.com/sharon-xa/pretty-alias/internal/highlight"
	"github.com/sharon-xa/pretty-alias/internal/shell"
	"github.com/sharon-xa/pretty-alias/internal/tui"
)

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1 // runtime failure: unreadable file, unsupported shell, ...
	ExitUsage = 2 // bad command line
)

const usage = `pretty-alias shows the aliases from your shell config in a scrollable,
syntax-highlighted table.

Usage:
  pretty-alias [flags]

Flags:
`

// Run is the whole program. It returns the process exit code.
// version is the build version ("dev" if unset).
func Run(args []string, stdout, stderr io.Writer, version string) int {
	fs := flag.NewFlagSet("pretty-alias", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		outs(stderr, usage)
		fs.PrintDefaults()
		outs(stderr, `
Environment:
  SHELL                Used to detect the shell when --shell is not given.
  NO_COLOR             Disable syntax highlighting.
  PRETTY_ALIAS_DEBUG   If set, write a debug log to the user cache dir
                       (see --help in the README for the exact path).

Exit codes: 0 ok (also when no aliases are found), 1 error, 2 bad usage.
`)
	}
	var (
		shellFlag = fs.String("shell", "", "shell dialect to parse: bash, zsh or fish (default: detect from $SHELL)")
		fileFlag  = fs.String("file", "", "config file to read (default: the shell's usual rc file)")
		plainFlag = fs.Bool("plain", false, "print name<TAB>command lines instead of the interactive table")
		versFlag  = fs.Bool("version", false, "print the version and exit")
	)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitOK
		}
		return ExitUsage
	}
	if fs.NArg() > 0 {
		outf(stderr, "pretty-alias: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return ExitUsage
	}
	if *versFlag {
		outf(stdout, "pretty-alias %s\n", Version(version))
		return ExitOK
	}

	closeLog := setupLogging()
	defer closeLog()

	sh, err := pickShell(*shellFlag)
	if err != nil {
		return fail(stderr, err)
	}

	path := *fileFlag
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fail(stderr, fmt.Errorf("cannot find your home directory: %w", err))
		}
		path = sh.ConfigFile(home)
	}
	slog.Debug("loading", "shell", sh, "file", path)

	aliases, err := alias.Load(sh, path)
	if err != nil {
		return fail(stderr, err)
	}
	slog.Debug("parsed", "aliases", len(aliases))

	if len(aliases) == 0 {
		outf(stderr, "pretty-alias: no aliases found in %s\n", tilde(path))
		return ExitOK
	}

	if *plainFlag || !isTerminal(stdout) {
		printPlain(stdout, aliases)
		return ExitOK
	}

	m := tui.New(aliases, highlight.New(sh), tilde(path))
	if err := tui.Run(m); err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

func pickShell(flagValue string) (shell.Shell, error) {
	if flagValue != "" {
		return shell.Parse(flagValue)
	}
	return shell.Detect(os.Getenv("SHELL"))
}

func fail(stderr io.Writer, err error) int {
	outf(stderr, "pretty-alias: %v\n", err)
	return ExitError
}

// printPlain writes one "name<TAB>command" line per alias, unstyled.
func printPlain(w io.Writer, aliases []alias.Alias) {
	for _, a := range aliases {
		outf(w, "%s\t%s\n", highlight.Sanitize(a.Name), highlight.Sanitize(a.Command))
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
}

// tilde shortens a path under $HOME to ~/... for display.
func tilde(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return path
}

// Version returns the version to display: the one injected at build time
// (-ldflags "-X main.version=..."), else the module version stamped by
// `go install pkg@version`, else "dev".
func Version(injected string) string {
	if injected != "" && injected != "dev" {
		return injected
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

// outf and outs write to a stream whose failure we can't do anything about
// (a closed stdout or stderr), so the error is deliberately discarded.
func outf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
func outs(w io.Writer, s string)                   { _, _ = fmt.Fprint(w, s) }
