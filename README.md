# pretty-alias

A small CLI that shows your shell aliases in a scrollable, syntax-highlighted table. Supports `bash`, `zsh` and `fish`.

## Features

- Detects your shell, reads its config file, and lists every alias with the command highlighted.
- Understands real alias syntax: quotes and escapes, several aliases per line, indented aliases, option flags (`alias -g`), trailing comments, and fish's `alias name 'cmd'` form.
- Never trusts the file: control characters are shown as symbols, never executed by your terminal.
- Plain `name<TAB>command` output for pipes and scripts.

## Installation

#### Via install script

```bash
curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh | sh
```

The script detects your OS (Linux, macOS) and CPU (amd64, arm64), downloads the matching release binary to a temporary directory, verifies it against the release's `checksums.txt`, checks that it really is an executable, and only then installs it. It installs to `/usr/local/bin` when run as root (use `| sudo sh` for a system-wide install) or when that directory is writable, otherwise to `~/.local/bin`, and never runs the downloaded binary.

| Variable | Effect |
|---|---|
| `PRETTY_ALIAS_VERSION` | Install a specific release, e.g. `v1.2.1` (default: the latest) |
| `INSTALL_DIR` | Install somewhere else, e.g. `INSTALL_DIR=$HOME/bin` |

For example: `curl -fsSL …/install.sh | INSTALL_DIR=$HOME/bin PRETTY_ALIAS_VERSION=v1.2.1 sh` (put the variables on the `sh` side of the pipe). If you'd rather not pipe to a shell, download `install.sh`, read it, and run it.

Releases up to v1.2.1 only published a Linux amd64 binary without checksums; the script still installs it, with a warning that it could not be verified. Newer releases must ship per-platform binaries and a `checksums.txt` (see `ARCHITECTURE.md`), and the script refuses to install a release that doesn't.

#### Via go install

```bash
go install github.com/sharon-xa/pretty-alias/v2@latest
```

The installed command is still `pretty-alias`. Note the `/v2`: the old path without it (`github.com/sharon-xa/pretty-alias@latest`) stays on v1.2.1.

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
| Mouse wheel | Scroll. The mouse is not captured, so you can still select and copy text; the terminal turns the wheel into `↑`/`↓` (most do by default, plain xterm doesn't). |
| `/` | Filter: type to show only aliases whose name or command contains the text (case-insensitive unless it has a capital letter). `enter` keeps the filter, `esc` clears it. While typing, every letter goes into the filter. |
| `esc` | Clear the filter |
| `q` / `ctrl+c` | Quit (`ctrl+c` also works while typing a filter) |

### Environment

| Variable | Effect |
|---|---|
| `SHELL` | Used to detect the shell when `--shell` is not given. |
| `NO_COLOR` | Disable syntax highlighting. |
| `PRETTY_ALIAS_DEBUG` | If set, write a debug log to `pretty-alias/debug.log` inside your user cache directory: `$XDG_CACHE_HOME` or `~/.cache` on Linux, `~/Library/Caches` on macOS. Never written to the current directory. |

### Current limitations

Only the one config file is read (`.bashrc`, `.zshrc` or `config.fish`), so aliases defined in files it `source`s (such as `~/.bash_aliases`) are not shown yet (see [#2](https://github.com/sharon-xa/pretty-alias/issues/2) and [#31](https://github.com/sharon-xa/pretty-alias/issues/31)). Aliases that span several lines are skipped.

## Development

Requires [Go](https://go.dev/dl/) (version in `go.mod`) and, optionally, [`just`](https://github.com/casey/just).

```sh
just            # list recipes
just test       # go test ./...
just check      # vet + lint + race tests + format check (what CI runs)
just fmt        # format (gofumpt + golines, 100 columns)
just smoke      # run every fixture through the real binary in a pty
just dist       # build release binaries with GoReleaser (snapshot) and test them with install.sh
just try testdata/zshrc zsh    # open the TUI on a fixture, without touching your real config
```

### Releasing

Push a version tag: `git tag v1.3.0 && git push origin v1.3.0`. The release workflow runs the tests, checks a snapshot build against `install.sh`, then publishes binaries for Linux and macOS (amd64, arm64) plus `checksums.txt` with [GoReleaser](https://goreleaser.com).

Layout and design rules are in [ARCHITECTURE.md](ARCHITECTURE.md); fixtures are documented in [testdata/README.md](testdata/README.md).

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for the TUI
- [Chroma](https://github.com/alecthomas/chroma) for syntax highlighting

## License

This project is licensed under the MIT License.
