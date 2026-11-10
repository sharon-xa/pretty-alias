# pretty-alias — Codebase Review

> **Status note (added after the restructure):** this is a point-in-time review of commit `6f50557`. The code layout it describes (`system/`, `table/`, `program.go`, `table.go`, `godotenv`, `.env`) no longer exists. The current design is in [ARCHITECTURE.md](ARCHITECTURE.md). Most of the bugs in §5 (panics, quoting, indentation, flags, comments, silent exits, resize crash, data race, `.env`) were fixed by the restructure and are covered by tests; the feature ideas in §7 are still open as GitHub issues.


_Reviewed 2026-11-10 at commit `6f50557` (branch `main`). Go 1.27.3 toolchain; `go build` and `go vet` pass._

Every file in the repo was read (about 480 lines of Go in total). Claims marked **[verified]** were reproduced by running the built binary in a pseudo-terminal against fixture rc files, or by a throwaway test in a scratch copy of the repo. Nothing in the repo was modified except this file.

---

## 1. Summary

`pretty-alias` is a small Bubble Tea TUI. It detects your shell from `$SHELL`, reads that shell's rc file, picks out lines starting with `alias`, and shows them in a scrollable, syntax-highlighted, two-column table (Alias / Command).

**What works well**

- It does one job, and the happy path looks good: Lip Gloss borders, Chroma/Monokai highlighting, a centred layout.
- It's small and easy to follow, and the `system` / `table` / TUI split is sensible.
- It handles the Bubble Tea "window size arrives after data" race correctly, and the alt-screen is used properly.
- Dependencies are modern and few.

**What needs attention**

- **Robustness:** several ordinary inputs make it crash or exit silently (§5).
- **Coverage:** it only reads one file per shell, and parsing is a `=`/space split, so real-world alias setups are often missed or mangled.
- **Project hygiene:** there are no tests, no CI and no release automation, and the installer has a real footgun (§6).
- **Interactivity:** the TUI is view-only, with very limited navigation and no help.

---

## 2. Folder structure

```
pretty-alias/
├── main.go            # entry point: .env + debug logging, starts tea.Program
├── program.go         # Bubble Tea model: Init / Update / View, scrolling
├── table.go           # (*model).GetTable(): builds the lipgloss table
├── system/
│   ├── shell.go       # GetUserShell(): parses $SHELL → "bash" | "zsh" | "fish"
│   └── aliases.go     # opens the rc file, returns raw "alias ..." lines
├── table/
│   ├── tableRows.go   # raw alias lines → [name, highlighted command] rows
│   └── style.go       # colour constants, Chroma highlighting, quote trimming
├── install.sh         # curl-to-/usr/local/bin installer, pinned to v1.2.1
├── TODO.md            # 4 items (3 open)
├── README.md, LICENSE (MIT)
├── .env               # TRACKED in git; contains PRETTY_ALIAS_DEBUG=true
├── .gitignore         # `pretty-alias`, `*.log`
└── go.mod / go.sum    # module github.com/sharon-xa/pretty-alias, go 1.24.1
```

**Observations**

- The layout is flat and readable. Everything in the root is `package main`, and `system/` and `table/` are real sub-packages.
- **Two different "table" things.** `table.go` (the TUI table builder) lives in `package main`, while `table/` is the package that builds the *rows*. `table.go` has to import both under an alias (`glossTable`). Rename one, for example `table/` → `rows/` or `parse/`, or move `GetTable` into the TUI package.
- **`table/` mixes concerns.** It holds colour constants, the highlighter, and alias parsing. Parsing is the part that deserves its own package (`alias` or `parse`), separate from rendering.
- **`system/` mixes concerns too.** It holds shell detection and file reading, and `table/` imports `system/` just to re-detect the shell. The shell should be detected once and passed in.
- **`.env` is committed.** See §6.
- **File naming.** `tableRows.go` is camelCase, which is unusual for Go. `table_rows.go` is idiomatic.

---

## 3. How the program flows

```
main()
 ├─ godotenv.Load(".env")                    # from CWD!
 ├─ PRETTY_ALIAS_DEBUG set? → append to ./debug.log, else log → io.Discard
 └─ tea.NewProgram(&model{}, WithAltScreen).Run()

Init()  → fetchAliases (a tea.Cmd, runs in a goroutine)
            ├─ system.GetAliases()   → read rc file, keep lines starting with "alias"
            ├─ table.GetTableRows()  → split name/command, trim quotes, highlight
            └─ writes m.rows directly, returns aliasesMsg

Update()
  aliasesMsg      → if size known: renderTable
  WindowSizeMsg   → store size, ready=true, renderTable
  tableMsg        → m.table = rendered string
  errMsg          → store error, tea.Quit
  KeyMsg          → q / ctrl+c quit; up/down adjust scrollOffset

View()  → split m.table into lines, slice [scrollOffset : scrollOffset+height],
          lipgloss.Place(centered, top)
```

The model is rendered as a single pre-built string that `View` slices by line. That's simple and works, but it's why scrolling can't be clamped properly (see §5, bug 3).

---

## 4. Usage

### CLI

There is **no CLI surface**: no flags, no subcommands, no arguments, no `--help`, no `--version`. `pretty-alias` just launches the TUI.

```sh
pretty-alias          # the only invocation
```

| Aspect | Current behaviour |
|---|---|
| Shell selection | Substring match on `$SHELL` (`fish`, `bash`, `zsh`). Any other shell makes it exit with code 1 and **no message**. |
| Source file | `bash` → `~/.bashrc`, `zsh` → `~/.zshrc`, `fish` → `~/.config/fish/config.fish`. Nothing else is read. |
| Env var | `PRETTY_ALIAS_DEBUG=<anything>` appends logs to `./debug.log`. Undocumented in the README. |
| Exit codes | 0 on quit and also on "no aliases found" (the message is never seen); 1 on startup errors. |
| Non-TTY use | Not supported. There is no plain-text output mode, so `pretty-alias \| grep` or scripting doesn't work. |

### TUI

| Key | Action |
|---|---|
| `↑` | Scroll up one line |
| `↓` | Scroll down one line |
| `q` / `Ctrl+C` | Quit |

- The README lists `q` / `Ctrl+C` / `↑` / `↓`, and that matches the code.
- **Layout:** a bordered table with a cyan bold header (`ALIAS`, `COMMAND`). The command column width is `terminal width − 20`, and rows are separated by a blank padded line. The block is centred horizontally.
- **Highlighting:** Chroma's *fish* lexer, Monokai style, 256-colour TTY formatter. It is used for every shell, including bash and zsh.
- **No on-screen help or footer.** Users have to know that `q` quits and the arrows scroll.
- **Missing common bindings:** `j`/`k`, `PgUp`/`PgDn`, `Home`/`End`, `g`/`G`, mouse wheel. There's no scroll indicator either, so you can't tell how much is below the fold.
- **Empty or error states:** the TUI quits and the alt-screen is torn down, so the user sees *nothing*. That's the open `TODO.md` item "Fix not showing anything when there's no aliases." **[verified]**

---

## 5. Bugs and robustness issues

Ordered by severity. Items marked **[verified]** were reproduced.

### Crashes (panics)

1. **Any line that starts with `a` and is shorter than 5 characters panics.** **[verified]**
   `system/aliases.go:90-93` checks `line[0] == 'a'` and then slices `line[:5]`. A line such as `a`, `ab`, `ack` or `awk` (common in rc files, e.g. a single-word command) panics with `slice bounds out of range [:5] with length N`.
   *Fix:* `strings.HasPrefix(strings.TrimSpace(line), "alias ")`.

2. **An alias with no value panics.** **[verified]**
   In the bash/zsh path, `alias foo` (or `alias -p`) gives `SplitN(..., "=", 2)` one element, and `aliasNameAndCommand[1]` is an index-out-of-range panic (`tableRows.go:72`). The fish path has the same problem. `alias` alone in fish panics with `slice bounds out of range [6:5]` because it does `alias[6:]` (`tableRows.go:31`), and `alias x` panics with `index out of range [1] with length 0` when it later indexes `[1]`.
   In the fish path, a name containing neither `=` nor a space leaves `aliasDeclerationType` as the zero rune, so `aliasNameAndCommand` stays `nil`.

3. **Widening or resizing the window after scrolling can panic in `View`.** **[verified]**
   `scrollOffset` is never clamped when `m.table` changes. Scrolling to the bottom of a narrow (wrapped, tall) table and then widening the window gives `slice bounds out of range [59:34]` at `program.go:124`. The `down` handler clamps against `m.height` but `WindowSizeMsg`/`tableMsg` don't re-clamp.
   *Fix:* clamp `scrollOffset` in `tableMsg` and `WindowSizeMsg`, or in `View`.

### Silent failures

4. **`log.Fatalln` is swallowed by the discarded logger.** **[verified]**
   `getConfigFile()` and `GetTableRows()` use `log.Fatalln`, but `main` sets the log output to `io.Discard` unless debug is on. A missing rc file or an unsupported shell (e.g. `SHELL=/bin/sh`) makes the process exit 1 with **no output at all**.
   These functions also `Fatalln` from inside a `tea.Cmd` goroutine, which bypasses Bubble Tea's terminal restore and the `errMsg` path that already exists. They should return errors instead.

5. **A nil dereference is possible when the shell isn't one of the three.** `getConfigFile` leaves `file` nil for an unknown shell and `err` nil from the earlier call, so `defer file.Close()` and `readFile(nil)` would be reached. In practice `GetUserShell` errors first, so this is latent rather than live.

### Wrong output

6. **Multiple aliases on one line are mangled.** **[verified]**
   `alias a="ls" b="pwd"` renders as name `a`, command `ls" b="pwd`.

7. **Indented aliases are ignored.** **[verified]** `  alias ll="ls"` is skipped because the check is `line[0] == 'a'`. This hits aliases inside `if`/`case` blocks, which are common in rc files. The result is "no aliases found".

8. **Alias flags become part of the name.** _(from reading the code, not run)_ In zsh, `alias -g L='| less'` splits on the first `=` and shows the name as `-g L`. The same happens for fish's `alias --save ll 'ls'`. The code never recognises option flags.

9. **Escaped or nested quotes are not understood.** `cleanAliasCommand` uses `strings.Trim`, which strips *all* leading and trailing quote characters, not just one pair. For example, `alias a='"hi"'` loses the inner quotes. Backslash escapes (`\"`, `'\''`) are not unescaped.

10. **Trailing comments aren't stripped.** `alias ll='ls -la' # list` keeps `# list` in the command column.

11. **bash/zsh highlighting uses the fish lexer.** It's a reasonable approximation, but a bash or zsh lexer would be more accurate for `$(...)`, `[[ ]]`, and so on.

### Concurrency and structure

12. **Data race on `m.rows`.** `fetchAliases` is a `tea.Cmd` that runs in its own goroutine and writes `m.rows` directly. Bubble Tea's model is meant to be touched only from `Update`. It works today only because the subsequent message orders the read. Return the rows in the message (`aliasesMsg` currently carries the raw strings that nobody reads) and assign in `Update`.

13. **`highlightCommand` prints to stdout and then dereferences nil.** On a missing lexer or style it calls `fmt.Println` (which would corrupt the alt-screen) and carries on. `lexer.Tokenise` on a nil lexer then panics. It also re-looks-up the lexer and style on every alias; hoist them to package level.

14. **A very narrow terminal gives a negative column width.** `style.Width(m.width - 20)` goes negative below 20 columns. `debug.log` shows it was tried down to width 39. Anything below 20 is untested.

15. **`Init`'s `if m.ready` guard is dead code.** `ready` is false at start, and `Init` is only called once.

16. **Logging logs stale values.** In the `WindowSizeMsg` case the log line prints `m.width`/`m.height` *before* assigning the new ones (the committed `debug.log` shows `Width: 0 , Height: 0` for the first message).

### Minor / style

- `readFile` reimplements `os.ReadFile` with a 1 KiB chunk loop. `defer clear(buf)`, `clear(content)`, `clear(aliases)` and `clear(lines)` appear to be "security" zeroing, but they don't do anything useful for a Go `string`. The `str := string(content)` copy has already made an unzeroed copy, and an alias list isn't a secret.
- `GetAliases` returns `*string` from `getConfigFile`, with no need for the pointer.
- Typos: `aliasDeclerationType`, `TrimedAlias`; exported-style local variable names.
- Error text begins with a capital letter and ends with `!` (`"Shell not found!"`), which Go style discourages.
- `GetUserShell` uses `strings.Contains` on `$SHELL`, so `/usr/bin/zsh-wrapper` or a path containing `fish` in a directory name would match. Use `filepath.Base`.
- `$SHELL` is the user's *login* shell, not necessarily the shell they're running (for example, running `zsh` from a bash login).
- `go.mod` declares `go 1.24.1`, which is fine, but `README` doesn't state a minimum Go version for `go install`.

---

## 6. Project hygiene, security and distribution

- **`.env` is committed** and contains `PRETTY_ALIAS_DEBUG=true`. `main` also calls `godotenv.Load(".env")` from the **current working directory** on every run. Consequences: (a) anyone running `pretty-alias` inside a project with its own `.env` has that file's variables loaded into the process and, if it sets `PRETTY_ALIAS_DEBUG`, a `debug.log` is written into that project; (b) the dev's debug flag is part of the repo. Drop `godotenv` entirely and read the real environment (`os.Getenv`). If you want a dev convenience, use `direnv` or a `Makefile` target, and gitignore `.env`.
- **`debug.log` is written to the CWD**, so it ends up in whatever directory you launched from. Use `os.UserCacheDir()` / `XDG_STATE_HOME`. `*.log` is gitignored, which is why `debug.log` isn't tracked.
- **`install.sh` has several problems:**
  - `curl -Lo` has no `-f`, so a 404 (for example a bad version) writes the error page to `/usr/local/bin/pretty-alias`, and the script then `chmod +x`'s it and prints "installed successfully!".
  - There is no checksum or signature verification of the downloaded binary.
  - `VERSION="1.2.1"` is hard-coded and has to be bumped by hand for each release.
  - It is not architecture-aware (the `linux-amd64` URL is commented out), so there is no macOS or arm64 handling; the asset is a bare `pretty-alias`.
  - It writes to `/usr/local/bin` and the README tells users to pipe it into `sudo sh`. Offer `PREFIX`/`~/.local/bin` as an alternative.
- **No tests.** `TODO.md` lists "Test the CLI with fish, zsh and bash", with 0, some and lots of aliases, and nothing automated exists. The parser (`createFishAliasRows`, `createBashZshAliasRows`, `cleanAliasCommand`) is pure and trivially table-testable, and every crash in §5 would have been caught by a test.
- **No CI** (`go vet`, `staticcheck`, `go test -race`) and **no release automation** (GoReleaser would also give you multi-arch binaries, checksums and Homebrew/AUR/Scoop packaging for free).
- **README gaps:** no screenshot or GIF, no mention of the debug env var, no list of supported rc files, and no note that only the one rc file is read.
- **TODO.md** has three open items; two are covered by §5/§7 below.

---

## 7. Recommended features

Grouped roughly by value for effort. "Effort" is a rough guess: S = hours, M = a day or two, L = more.

### A. Fix-first (not features, but they unblock the rest)

1. **Robust alias parser.** Replace the line splitting with a small tokenizer that handles leading whitespace, `alias -g`/`alias -s` (zsh), multiple definitions per line, `=` vs space (fish), quoted values with escapes, and trailing comments. Return `[]Alias{Name, Command, Source, Line}` rather than `[][]string`. _(M)_
2. **Real error and empty states in the TUI.** Show "No aliases found in `~/.zshrc`" on screen (and keep it until a key is pressed). Return non-zero exit codes with a message on stderr for startup errors. _(S)_
3. **Table-driven tests + fixtures** for the parser and a `teatest` smoke test for the model. _(S–M)_

### B. Finding aliases in more places

4. **Follow `source` / `.` includes** and read the common alias files: `~/.bash_aliases`, `~/.zsh_aliases`, `~/.zshrc.d/*`, `~/.config/fish/conf.d/*.fish`, `~/.oh-my-zsh/custom/*.zsh`. Show a "Source" column (file:line). _(M)_
5. **Fish `abbr` and functions.** Show `abbr -a` abbreviations and optionally `functions/*.fish`, since many fish users don't use `alias` at all. _(M)_
6. **Live/effective aliases.** Run `$SHELL -i -c 'alias'` (bash/zsh) or `fish -c 'alias'` to list what the shell actually has loaded, including plugin-provided aliases (oh-my-zsh, starship, etc.), with a flag to toggle between "file" and "live" mode. _(M)_
7. **More shells.** nushell, PowerShell, xonsh, elvish. Nushell aliases (`alias ll = ls -l`) would need their own parser. _(M each)_
8. **`--file <path>` / `--shell <name>` flags** to inspect a specific file or override detection (also makes testing and demos easy). _(S)_

### C. TUI usability

9. **Search / filter** with `/` (live substring or fuzzy match on name and command), using Bubbles' `textinput`. This is probably the single most valuable addition for people with 100+ aliases. _(S–M)_
10. **Proper navigation:** `j`/`k`, `PgUp`/`PgDn`, `Ctrl+u`/`Ctrl+d`, `g`/`G`, `Home`/`End`, mouse wheel, plus a scroll percentage indicator. Switch to the Bubbles `viewport` or `table` components, which handle clamping and resize for you and would remove bug 3. _(S)_
11. **Help footer** (`?` toggles Bubbles' `help` view) showing the key bindings. _(S)_
12. **Selection and copy.** Move a cursor row-by-row and press `Enter`/`y` to copy the alias *name* or *command* to the clipboard (OSC52, which is already in the dependency tree via `go-osc52`, so it works over SSH). _(S–M)_
13. **Sort and group toggles:** alphabetical, by source file, by first word of command (e.g. all `git` aliases together). _(S)_
14. **Responsive layout.** Truncate or wrap sensibly on narrow terminals, and size the alias column to its content instead of the fixed `width − 20` command width. _(S)_
15. **Theming.** Selectable Chroma style (`--theme`), light-terminal support (detect background with `lipgloss.HasDarkBackground`) and honouring `NO_COLOR`. Pick the lexer per shell (`bash`/`zsh`/`fish`). _(S)_

### D. Categories and organisation (already on the TODO)

16. **Category syntax in the rc file.** For example a `# pa: Git` section comment, or a `# [group: docker]` marker above a block of aliases, rendered as section headers in the table. Reuse the comment-parsing work from item 1. _(M)_
17. **Per-alias descriptions** via a trailing comment (`alias gs='git status' # Short status`), shown as a third column or on a detail line. _(S, once comments are parsed)_
18. **Detect problems:** duplicate alias names, aliases that shadow real commands in `$PATH` (e.g. `alias ls=...`, which is often intentional, so show it as info), and aliases whose target command doesn't exist. _(M)_

### E. Scripting-friendly CLI

19. **Non-interactive output when stdout isn't a TTY** (or with `--plain`): `--json`, `--tsv`, and a simple colourised `--print` that dumps the table once without the alt-screen. Makes `pretty-alias | grep git` and shell-completion/fzf integration possible. _(S–M)_
20. **Standard CLI hygiene:** `--help`, `--version` (ldflags-injected), and shell completions. Use `cobra` or `urfave/cli` only if more subcommands are planned; the stdlib `flag` package is enough for a handful of flags. _(S)_
21. **fzf-style picker mode** (`pretty-alias pick`) that prints the selected alias name/command to stdout, so it can be bound to a key in the shell. _(M)_

### F. Management (bigger, more opinionated)

22. **Add / edit / delete aliases from the TUI**, writing back to the correct file with a backup and a diff preview. High value but carries risk of corrupting rc files, so it should be explicit and behind a flag at first. _(L)_
23. **Alias usage stats:** read shell history (`~/.bash_history`, `~/.zsh_history` extended format, fish history) to show how often each alias is used and surface never-used ones. _(M)_
24. **Suggest aliases:** the reverse of 23, finding frequently typed long commands in history that deserve an alias. _(M)_
25. **Export / sync:** export to a single portable file or convert between shell syntaxes (bash ⇄ fish). _(M)_

### G. Distribution

26. **GoReleaser + GitHub Actions** for tagged releases: Linux/macOS/Windows × amd64/arm64, checksums, and `brew`, AUR and `scoop` packaging. Rework `install.sh` to detect OS/arch, use `curl -fsSL`, verify the checksum and install to a user-writable prefix if not root. _(M)_

---

## 8. Suggested order of work

1. §5 bugs 1–4 (panics and silent exits). They are small, mechanical fixes.
2. Parser rewrite with tests (A1, A3). Everything in B, C and D builds on a trustworthy parser.
3. Replace the hand-rolled scroll with Bubbles `viewport` and add the help footer (C10, C11), which also removes bug 3.
4. Remove `godotenv` and the committed `.env`; move the debug log to a cache/state dir (§6).
5. Search/filter (C9), then source-file column and `source` following (B4).
6. `--plain`/`--json` output and `--help`/`--version` (E19, E20).
7. Release pipeline and a safer `install.sh` (G26).
8. Categories and descriptions (D16, D17) to close out `TODO.md`.
