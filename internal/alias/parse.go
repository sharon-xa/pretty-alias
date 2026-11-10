package alias

import (
	"os"
	"strings"

	"github.com/sharon-xa/pretty-alias/internal/shell"
)

// Parse extracts alias definitions from the text of a shell config file.
// file is only recorded in the results (it is not read).
//
// Parse never fails and never panics: lines it can't understand (no value,
// unterminated quote, ...) are skipped. What it handles:
//
//   - leading whitespace, CRLF line endings and a UTF-8 BOM
//   - several definitions on one line (alias a=1 b=2)
//   - option flags (bash -p, zsh -g/-s/-L, fish --save) and "--"
//   - quotes and escapes ('..', "..", \x, $'..'), resolved like the shell does
//   - trailing comments, returned in Alias.Comment
//   - fish's "alias name 'cmd'" form as well as "alias name=cmd"
//
// Known limits: definitions that span several lines, and aliases created by
// anything other than a line starting with the word "alias".
func Parse(sh shell.Shell, src, file string) []Alias {
	src = strings.TrimPrefix(src, utf8BOM)
	fish := sh == shell.Fish

	var out []Alias
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSuffix(line, "\r")
		rest, ok := afterKeyword(strings.TrimLeft(line, " \t"), "alias")
		if !ok {
			continue
		}
		words, comment, ok := splitWords(rest, fish)
		if !ok {
			continue
		}

		var defs []definition
		if fish {
			defs = fishDefinitions(words)
		} else {
			defs = posixDefinitions(words)
		}
		for _, d := range defs {
			out = append(out, Alias{
				Name:    d.name,
				Command: d.command,
				Comment: comment,
				File:    file,
				Line:    i + 1,
			})
		}
	}
	return out
}

// Load reads path and parses it for the given shell.
func Load(sh shell.Shell, path string) ([]Alias, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(sh, string(b), path), nil
}

// utf8BOM is the byte order mark some Windows editors put at the start of a file.
const utf8BOM = "\xef\xbb\xbf"

type definition struct{ name, command string }

// afterKeyword reports whether line starts with the whole word kw and returns
// the text after it ("aliasing=1" and "alia" don't match).
func afterKeyword(line, kw string) (string, bool) {
	if !strings.HasPrefix(line, kw) {
		return "", false
	}
	rest := line[len(kw):]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", false
	}
	return rest, true
}

// posixDefinitions interprets the arguments of bash/zsh "alias".
// Every argument that looks like name=value defines an alias; flags and
// bare names (which only query an alias) are skipped.
func posixDefinitions(words []string) []definition {
	var defs []definition
	optionsDone := false
	for _, w := range words {
		if !optionsDone {
			if w == "--" {
				optionsDone = true
				continue
			}
			if len(w) > 1 && w[0] == '-' {
				continue // -p, -g, -s, -L, -m, ...
			}
		}
		if i := strings.IndexByte(w, '='); i > 0 {
			defs = append(defs, definition{name: w[:i], command: w[i+1:]})
		}
	}
	return defs
}

// fishDefinitions interprets the arguments of fish "alias":
//
//	alias [options] NAME DEFINITION...
//	alias [options] NAME=DEFINITION
func fishDefinitions(words []string) []definition {
	i := 0
options:
	for i < len(words) {
		w := words[i]
		switch {
		case w == "--":
			i++
			break options
		case w == "-d" || w == "--description":
			i += 2 // option with a value
		case len(w) > 1 && w[0] == '-':
			i++ // -s, --save, -h, --description=...
		default:
			break options
		}
	}
	if i >= len(words) {
		return nil
	}
	rest := words[i:]

	if len(rest) == 1 { // alias NAME=DEFINITION
		if j := strings.IndexByte(rest[0], '='); j > 0 {
			return []definition{{name: rest[0][:j], command: rest[0][j+1:]}}
		}
		return nil // "alias name" with no definition
	}
	if strings.Contains(rest[0], "=") {
		return nil // fish rejects this: names can't contain '='
	}
	return []definition{{name: rest[0], command: strings.Join(rest[1:], " ")}}
}
