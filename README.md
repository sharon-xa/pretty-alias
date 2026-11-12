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
| `PRETTY_ALIAS_NO_COMPLETIONS` | Set to anything to skip installing [shell completion](#shell-completion) |

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
pretty-alias --plain --sort name    # alphabetical
pretty-alias --theme dracula        # pick the highlighting theme
pretty-alias --version
pretty-alias --completion zsh       # print a shell completion script
```

| Flag | Meaning |
|---|---|
| `--shell bash\|zsh\|fish` | Dialect to parse. Default: detected from `$SHELL`. |
| `--file PATH` | Config file to read. Default: `~/.bashrc`, `~/.zshrc` or `~/.config/fish/config.fish`. |
| `--plain` | Print `name<TAB>command` lines instead of the TUI. Used automatically when stdout is not a terminal. |
| `--sort file\|name\|command` | Order of the aliases: as defined in the file (default), by name, or by command (which puts all the `git …` aliases together). Case is ignored. Also applies to `--plain`. |
| `--theme NAME` | Syntax highlighting theme. Default `auto`: `monokai`, or `monokailight` when the terminal reports a light background. Any [Chroma style](https://xyproto.github.io/splash/docs/) works, e.g. `dracula`, `nord`, `github`; an unknown name prints the full list. |
| `--version` | Print the version. |
| `--completion bash\|zsh\|fish` | Print a completion script for that shell (see below). |

Exit codes: `0` success (also when no aliases are found, which is reported on stderr), `1` error (unreadable file, unsupported shell), `2` bad usage.

### Shell completion

Tab completes the flags and their values: shells, sort orders, theme names, and file paths for `--file`.

**With the install script it is set up for you.** For each of bash, zsh and fish that is installed, the script writes a small file where that shell looks for completions by itself. The file asks `pretty-alias --completion <shell>` for the real script when you first press tab, so it stays current when you upgrade. Open a new shell after installing.

| Shell | File (as root: under `/usr/local/share`) | Needs |
|---|---|---|
| bash | `~/.local/share/bash-completion/completions/pretty-alias` | the `bash-completion` package (installed by default on most Linux distributions) |
| zsh | `~/.local/share/zsh/site-functions/_pretty-alias`, or a writable `site-functions` directory already in zsh's `$fpath` | if it's the first one, a line in `~/.zshrc` (below) |
| fish | `~/.local/share/fish/vendor_completions.d/pretty-alias.fish` | nothing |

zsh has no per-user directory it always reads, so unless one is already set up, the installer prints this line for you to add to `~/.zshrc`, **above** the line that runs `compinit` (with oh-my-zsh: above `source $ZSH/oh-my-zsh.sh`):

```zsh
fpath=(~/.local/share/zsh/site-functions $fpath)
```

**With `go install`** (or to set it up by hand), add one line to your shell's config:

```sh
eval "$(pretty-alias --completion bash)"            # ~/.bashrc
source <(pretty-alias --completion zsh)             # ~/.zshrc, after compinit
pretty-alias --completion fish | source             # ~/.config/fish/config.fish
```

### Keys

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Select the previous / next alias (the view follows) |
| `pgup` / `b`, `pgdn` / `f` / `space` | Scroll one page (the selection comes along) |
| `ctrl+u`, `ctrl+d` | Scroll half a page |
| `g` / `home`, `G` / `end` | First / last alias |
| `y` / `enter` | Copy the selected alias's command to the clipboard |
| `Y` | Copy the selected alias's name |
| `s` | Sort: cycle file order → by name → by command (the footer shows the order unless it's file order) |
| Mouse wheel | Move the selection. The mouse is not captured, so you can still select and copy text; the terminal turns the wheel into `↑`/`↓` (most do by default, plain xterm doesn't). |
| `/` | Filter: type to show only aliases whose name or command contains the text (case-insensitive unless it has a capital letter). `enter` keeps the filter, `esc` clears it. While typing, every letter goes into the filter. |
| `?` | Show or hide every key binding (the footer only lists `/`, `y`, `?` and `q`) |
| `esc` | Close the `?` help, or else clear the filter |
| `q` / `ctrl+c` | Quit (`ctrl+c` also works while typing a filter) |

Copying uses OSC 52, so it works over SSH too, but not every terminal supports it. Inside tmux it needs `set -g set-clipboard on` in `~/.tmux.conf` (tmux's default, `external`, drops it). The text is copied as shown: control characters in a command appear as symbols such as `␊`, so pasting it can't run anything by surprise.

### Environment

| Variable | Effect |
|---|---|
| `SHELL` | Used to detect the shell when `--shell` is not given. |
| `NO_COLOR` | Disable syntax highlighting. |
| `PRETTY_ALIAS_DEBUG` | If set, write a debug log to `pretty-alias/debug.log` inside your user cache directory: `$XDG_CACHE_HOME` or `~/.cache` on Linux, `~/Library/Caches` on macOS. Never written to the current directory. |

### Current limitations

Only the one config file is read (`.bashrc`, `.zshrc` or `config.fish`), so aliases defined in files it `source`s (such as `~/.bash_aliases`) are not shown yet (see [#2](https://github.com/sharon-xa/pretty-alias/issues/2) and [#31](https://github.com/sharon-xa/pretty-alias/issues/31)). Aliases that span several lines are skipped.

The background (light or dark) is detected by asking the terminal when the program starts. Terminals that don't answer get the dark colors; on a light one, pass `--theme monokailight` (the UI's own colors still assume dark then).

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
