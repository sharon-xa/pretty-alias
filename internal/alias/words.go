package alias

import "strings"

// splitWords splits s into shell words, resolving quotes and backslash
// escapes the way a shell would. It stops (without error) at the first
// unquoted comment or command separator (; & |) and returns the comment text.
//
// ok is false when a quote is left open, so the caller can skip the line
// instead of guessing.
//
// fish selects fish's escape rules inside single quotes (\\ and \' are
// escapes there; everywhere else in sh/bash/zsh a single quote is fully
// literal).
func splitWords(s string, fish bool) (words []string, comment string, ok bool) {
	var (
		cur    strings.Builder
		inWord bool
	)
	flush := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			flush()
			i++

		case c == '#' && !inWord:
			return words, strings.TrimSpace(s[i+1:]), true

		case c == ';' || c == '&' || c == '|':
			flush()
			return words, "", true

		case c == '\\':
			inWord = true
			if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i += 2
			} else { // trailing backslash (line continuation): ignore
				i++
			}

		case c == '\'':
			inWord = true
			end, text, closed := scanSingle(s, i+1, fish)
			if !closed {
				return nil, "", false
			}
			cur.WriteString(text)
			i = end

		case c == '"':
			inWord = true
			end, text, closed := scanDouble(s, i+1)
			if !closed {
				return nil, "", false
			}
			cur.WriteString(text)
			i = end

		case c == '$' && i+1 < len(s) && s[i+1] == '\'':
			inWord = true
			end, text, closed := scanANSIC(s, i+2)
			if !closed {
				return nil, "", false
			}
			cur.WriteString(text)
			i = end

		default:
			inWord = true
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return words, "", true
}

// scanSingle reads a single-quoted string whose opening quote is at start-1.
// It returns the index just past the closing quote.
func scanSingle(s string, start int, fish bool) (end int, text string, closed bool) {
	var b strings.Builder
	for i := start; i < len(s); i++ {
		switch {
		case s[i] == '\'':
			return i + 1, b.String(), true
		case fish && s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\'' || s[i+1] == '\\'):
			b.WriteByte(s[i+1])
			i++
		default:
			b.WriteByte(s[i])
		}
	}
	return len(s), "", false
}

// scanDouble reads a double-quoted string whose opening quote is at start-1.
// Inside double quotes a backslash only escapes $ ` " \ (and newline).
func scanDouble(s string, start int) (end int, text string, closed bool) {
	var b strings.Builder
	for i := start; i < len(s); i++ {
		switch {
		case s[i] == '"':
			return i + 1, b.String(), true
		case s[i] == '\\' && i+1 < len(s) && strings.IndexByte("$`\"\\", s[i+1]) >= 0:
			b.WriteByte(s[i+1])
			i++
		default:
			b.WriteByte(s[i])
		}
	}
	return len(s), "", false
}

// scanANSIC reads a bash/zsh $'...' string (opening quote at start-1),
// resolving the common escapes.
func scanANSIC(s string, start int) (end int, text string, closed bool) {
	var b strings.Builder
	for i := start; i < len(s); i++ {
		switch {
		case s[i] == '\'':
			return i + 1, b.String(), true
		case s[i] == '\\' && i+1 < len(s):
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case 'e', 'E':
				b.WriteByte(0x1b)
			case 'a':
				b.WriteByte(0x07)
			default: // \\ \' \" and anything unknown: the character itself
				b.WriteByte(s[i])
			}
		default:
			b.WriteByte(s[i])
		}
	}
	return len(s), "", false
}
