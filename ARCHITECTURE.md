# Architecture

`pretty-alias` reads a shell config file, finds the alias definitions, and shows them. Everything else follows from keeping those three steps separate.

```
main.go                     thin: os.Exit(app.Run(...)). Lives in the module root so that
                            `go install github.com/sharon-xa/pretty-alias/v2@latest` works.
internal/
  app/        composition root: flags, wiring, exit codes, logging, plain output,
              shell completion scripts
  shell/      Shell type (bash|zsh|fish), detection from $SHELL, rc-file paths
  alias/      Alias type, shell-word tokenizer, per-dialect parsers, file loading
  highlight/  Chroma wrapper: lexer per shell, Sanitize() for untrusted text
  tui/        Bubble Tea model, key map, table layout
testdata/     fixture rc files, a generator for big ones, README with expectations
```

## Data flow

```
$SHELL / --shell ──► shell.Shell ──┐
                                   ├──► alias.Load ──► []alias.Alias ──┬──► plain output (stdout)
--file / ~/.zshrc ─────────────────┘                                   └──► tui.New(…, highlight) ──► tui.Run
```

Loading happens **before** the terminal is taken over. That is deliberate:

- errors are printed to stderr with an exit code instead of vanishing with the alt-screen;
- the TUI only ever receives valid data, so it has no loading or error states;
- there is no goroutine writing into the model, so no data races.

## Dependency rules

```
app ──► tui ──► highlight ──► shell
 │       │
 │       └────► alias ──► shell
 └──► alias, shell, highlight
```

- `shell` and `alias` know nothing about terminals, colours or Bubble Tea. They take text in and return values; this is why they are fully unit-tested against `testdata/`.
- `tui` never reads files, never calls `os.Getenv`, never exits the process. It gets `[]alias.Alias` and returns when the user quits.
- Only `app` may print to stdout/stderr, read the environment, or decide an exit code.
- Nothing outside `app` calls `log.Fatal`, `os.Exit` or `panic`. Functions return `error`.
- Untrusted text (alias names and commands come from a file) must go through `highlight.Sanitize` before it is printed; `tui` and the plain output already do.

## Where new features go

| Feature | Where |
|---|---|
| Follow `source` includes, more files (#31) | `alias.Load` returns aliases from several files; `Alias.File`/`Line` already exist |
| Fish `abbr`, more shells (#33, #34) | new `shell.Shell` value + a `…Definitions` function in `alias/parse.go` |
| Descriptions / categories (#38, #39) | `Alias.Comment` is already parsed; add a field and a column in `tui/table.go` |
| Search / filter (#25) | Done (substring, smart-case): `tui.Model.filter` + `filterRows`. Fuzzy matching would replace `filterRows` |
| `--json`, `--tsv` (#36) | next to `printPlain` in `app`; the data is already typed |
| Copy to clipboard, selection (#28) | Done: `tui/cursor.go`. `renderTable` returns each row's line span; the cursor is drawn in the viewport's gutter, so moving it never re-renders the table |
| Sorting (#29) | Done: `alias.Order` (`Sort`, `Key`) shared by `--sort` for `--plain` and the TUI's `SetOrder` (`tui/order.go`). Grouping under headings belongs with categories (#38); grouping by file with #31 |
| Theme / colors (#30) | Done: `--theme` in `app`, themes in `highlight` (`AutoTheme` uses Chroma's light/dark counterpart pairs), UI palette in `tui/table.go` (`newPalette`), detection in `tui/colors.go` |
| CLI flags, shell completion (#35) | Done: flags in `app.Run` (stdlib `flag`). `--completion` (`app/completion.go`) builds the bash/zsh/fish scripts from the `FlagSet` itself; a new flag with fixed values only needs an entry in `completionValues` |
| New key bindings | `tui/keys.go`: add it to `keyMap` and `FullHelp()` (the `?` view, 4 columns that fit in 80). `ShortHelp()` (the footer) stays at `/`, `y`, `?`, `q` so it fits in 60 columns |
| Diagnostics: duplicates, shadowing (#40) | new package `internal/lint` taking `[]alias.Alias` |
| Add/edit/delete (#43) | new package `internal/edit`, using `Alias.File`/`Line`; `tui` only emits intents |

## Testing

- `internal/alias`: table tests for every parsing rule, plus every fixture in `testdata/` with its expected row count, plus a panic fuzz over truncations of tricky lines.
- `internal/tui`: drives `Update`/`View` directly with synthetic messages, including the resize-while-scrolled regression and hostile content.
- `internal/app`: runs the whole program (flags → file → output) with `--plain`; includes "every fixture runs without crashing".
- `install_test.go` (repo root) runs the real `install.sh` against a local fake release server: verified install, 404, checksum mismatch, missing checksums, an HTML error page, atomic re-install, and the legacy single-asset release.
- `just smoke` runs the real binary in a pty over every fixture.
- `just dist` (and the `release-dry-run` CI job) builds a GoReleaser snapshot and runs `scripts/check-dist.sh`, which checks the assets against the release contract and installs them with the real `install.sh` from a local HTTP server.
- Regenerate big fixtures with `just fixtures`; they are deterministic.

## Decisions worth remembering

- **`main.go` stays in the root**, not `cmd/pretty-alias/`, so the documented `go install …@latest` path keeps working.
- **Parsing is its own package with a typed result** (`alias.Alias`), not `[][]string`. Everything later (search, JSON, sources) depends on that.
- **No `.env`/`godotenv`.** Configuration is the real environment plus flags. The debug log goes to the user cache dir, never the working directory.
- **`esc` never quits.** `q` and `ctrl+c` quit; `esc` closes the `?` help if it is open, otherwise it belongs to the filter (cancel while typing, clear when applied). While the filter input has focus every printable key is text, so only `enter`, `esc` and `ctrl+c` are commands there.
- **The footer keeps its keys visible.** When the line is too long, the status (count, path, percentage) gives way first, and the path is shortened from the left.
- **Mouse is not captured**, so you can select and copy text with the terminal as usual. The wheel still scrolls: on the alternate screen most terminals send `↑`/`↓` for it when the app doesn't capture the mouse ("alternate scroll"; plain xterm is the exception), and with the cursor (#28) it moves the selection. There is no click-to-select; that would need mouse capture.
- **Copying is OSC 52 only** (`tea.SetClipboard`), so `tui` does no I/O and copying works over SSH. Terminals never confirm OSC 52, so the "copied" toast can't know it worked; tmux needs `set-clipboard on`. The copied text is the sanitized text (what is on screen), because a raw newline from a config file would run a command when pasted. A fallback to `wl-copy`/`xclip`/`pbcopy` would have to be injected by `app`.
- **The background color is the one thing settled after the TUI starts.** `Init` asks the terminal (`tea.RequestBackgroundColor`), and the first frame waits for the answer for at most 150 ms (`tui/colors.go`); a light answer switches the UI palette and re-highlights the commands once with the light highlighter `app` passed to `DetectBackground`. Querying before the TUI (`lipgloss.HasDarkBackground`) was tried and rejected: where nothing answers (`just smoke`, some CI ptys) it blocks ~4 s and swallows the keys typed meanwhile. With no colors (`NO_COLOR`) nothing is asked.
- **Highlighting is done once, up front** (`tui.New`), plus once more on a light background. That costs ~0.3 s for 2000 aliases and ~0.03 s for a typical 200. If this ever matters, highlight lazily for the visible rows. The same goes for rendering: each filter keystroke re-renders the matching rows (~0.1 s at 2000 matches, ~0.01 s at 200); render only the visible rows if that ever feels slow.
- **Release asset contract** (what `install.sh` expects): each release publishes `pretty-alias-<os>-<arch>` for `linux`/`darwin` × `amd64`/`arm64`, plus a `checksums.txt` in `sha256sum` format (`<hash>  <filename>`). GoReleaser (`.goreleaser.yaml`) produces exactly this: bare binaries (`formats: [binary]`) and a fixed checksum file name, both of which differ from GoReleaser's defaults. `scripts/check-dist.sh` enforces it in CI and before every release. The one exception is the legacy v1.2.1 release (a bare `pretty-alias` Linux amd64 binary with no checksums), which the installer still accepts with a warning.
- **Releases are tag-driven.** Pushing a `v*` tag runs `.github/workflows/release.yml`: tests, a snapshot build plus the contract check, then `goreleaser release`. The binary's `--version` is the tag (`main.version` via `-ldflags`).
- **`go` directive follows the latest release** (see `go.mod`); `just upgrade` bumps dependencies.
