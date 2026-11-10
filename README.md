# pretty-alias

A small CLI that shows your shell aliases in a scrollable, syntax-highlighted table. Supports `bash`, `zsh` and `fish`.

## Features

- Detects your shell, reads its config file, and lists every alias with the command highlighted.
- Understands real alias syntax: quotes and escapes, several aliases per line, indented aliases, option flags (`alias -g`), trailing comments, and fish's `alias name 'cmd'` form.
- Never trusts the file: control characters are shown as symbols, never executed by your terminal.
- Plain `name<TAB>command` output for pipes and scripts.

## Installation

#### Via bash script

```bash
curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh | sudo sh
```

#### Via go install

```bash
go install github.com/sharon-xa/pretty-alias@latest
```

## Usage

```sh
pretty-alias                         # interactive table for your current shell
pretty-alias --shell zsh             # force a shell instead of reading $SHELL
pretty-alias --file ~/dotfiles/aliases.zsh --shell zsh
pretty-alias --plain | grep git      # no TUI: one "name<TAB>command" line per alias
pretty-alias --version
```

| Flag | Meaning |
|---|---|
| `--shell bash\|zsh\|fish` | Dialect to parse. Default: detected from `$SHELL`. |
| `--file PATH` | Config file to read. Default: `~/.bashrc`, `~/.zshrc` or `~/.config/fish/config.fish`. |
| `--plain` | Print `name<TAB>command` lines instead of the TUI. Used automatically when stdout is not a terminal. |
| `--version` | Print the version. |

Exit codes: `0` success (also when no aliases are found, which is reported on stderr), `1` error (unreadable file, unsupported shell), `2` bad usage.

### Keys

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Scroll one line |
| `pgup` / `b`, `pgdn` / `f` / `space` | Scroll one page |
| `ctrl+u`, `ctrl+d` | Scroll half a page |
| `g` / `home`, `G` / `end` | Top / bottom |
| `q` / `esc` / `ctrl+c` | Quit |

### Environment

| Variable | Effect |
|---|---|
| `SHELL` | Used to detect the shell when `--shell` is not given. |
| `NO_COLOR` | Disable syntax highlighting. |
| `PRETTY_ALIAS_DEBUG` | If set, write a debug log to `$XDG_CACHE_HOME/pretty-alias/debug.log` (`~/.cache/pretty-alias/debug.log` on Linux). Never written to the current directory. |

### Current limitations

Only the one config file is read (`.bashrc`, `.zshrc` or `config.fish`), so aliases defined in files it `source`s (such as `~/.bash_aliases`) are not shown yet (see [#2](https://github.com/sharon-xa/pretty-alias/issues/2) and [#31](https://github.com/sharon-xa/pretty-alias/issues/31)). Aliases that span several lines are skipped.

## Development

Requires [Go](https://go.dev/dl/) (version in `go.mod`) and, optionally, [`just`](https://github.com/casey/just).

```sh
just            # list recipes
just test       # go test ./...
just check      # vet + lint + race tests + format check (what CI runs)
just smoke      # run every fixture through the real binary in a pty
just try testdata/zshrc zsh    # open the TUI on a fixture, without touching your real config
```

Layout and design rules are in [ARCHITECTURE.md](ARCHITECTURE.md); fixtures are documented in [testdata/README.md](testdata/README.md).

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for the TUI
- [Chroma](https://github.com/alecthomas/chroma) for syntax highlighting

## License

This project is licensed under the MIT License.
