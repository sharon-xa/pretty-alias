# Architecture

`pretty-alias` reads a shell config file, finds the alias definitions, and shows them. Everything else follows from keeping those three steps separate.

```
main.go                     thin: os.Exit(app.Run(...)). Lives in the module root so that
                            `go install github.com/sharon-xa/pretty-alias@latest` works.
internal/
  app/        composition root: flags, wiring, exit codes, logging, plain output
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
| Search / filter (#25) | `tui`: a text input in `Model`, filter `rows` before `renderTable` |
| `--json`, `--tsv` (#36) | next to `printPlain` in `app`; the data is already typed |
| Copy to clipboard, selection (#28) | `tui`: a cursor in `Model` + a binding in `keys.go` |
| Diagnostics: duplicates, shadowing (#40) | new package `internal/lint` taking `[]alias.Alias` |
| Add/edit/delete (#43) | new package `internal/edit`, using `Alias.File`/`Line`; `tui` only emits intents |

## Testing

- `internal/alias`: table tests for every parsing rule, plus every fixture in `testdata/` with its expected row count, plus a panic fuzz over truncations of tricky lines.
- `internal/tui`: drives `Update`/`View` directly with synthetic messages, including the resize-while-scrolled regression and hostile content.
- `internal/app`: runs the whole program (flags → file → output) with `--plain`; includes "every fixture runs without crashing".
- `just smoke` runs the real binary in a pty over every fixture.
- Regenerate big fixtures with `just fixtures`; they are deterministic.

## Decisions worth remembering

- **`main.go` stays in the root**, not `cmd/pretty-alias/`, so the documented `go install …@latest` path keeps working.
- **Parsing is its own package with a typed result** (`alias.Alias`), not `[][]string`. Everything later (search, JSON, sources) depends on that.
- **No `.env`/`godotenv`.** Configuration is the real environment plus flags. The debug log goes to the user cache dir, never the working directory.
- **Mouse is not captured**, so you can select and copy text with the terminal as usual. Revisit if/when row selection (#28) lands.
- **Highlighting is done once, up front** (`tui.New`). That costs ~0.3 s for 2000 aliases and ~0.03 s for a typical 200. If this ever matters, highlight lazily for the visible rows.
- **`go` directive follows the latest release** (see `go.mod`); `just upgrade` bumps dependencies.
