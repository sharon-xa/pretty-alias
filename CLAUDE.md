# CLAUDE.md

`pretty-alias` reads a shell rc file (bash, zsh, fish), parses the alias definitions, and shows them in a scrollable, syntax-highlighted TUI, or as `name<TAB>command` lines with `--plain` or when stdout isn't a terminal. Go, Bubble Tea v2, Lip Gloss v2, Chroma.

Read [ARCHITECTURE.md](ARCHITECTURE.md) before structural changes. It has the package map, data flow, dependency rules, where each planned feature goes, and the decisions not to revisit.

## Commands

```sh
just check     # vet + golangci-lint (incl. gofumpt/golines format check) + race tests (what CI runs)
just smoke     # real binary in a pty over every fixture; everything must be "ok" or "EMPTY"
just fmt       # apply gofumpt + golines (config in .golangci.yml, max line length 100)
just test      # go test ./...
just try testdata/zshrc zsh            # open the TUI on a fixture in a throwaway $HOME
just plain testdata/config.fish fish   # parsed output, no TUI
just fixtures  # regenerate testdata/large (deterministic; only the default n/seed is committed)
just dist      # GoReleaser snapshot into dist/ + scripts/check-dist.sh (installs it with install.sh)
```

Run one test: `go test -run TestScrolling ./internal/tui`.

## Definition of done

A change is done only when **`just check` and `just smoke` both pass**. If either fails, show the output; don't say it's finished.

## Architecture rules (summary; details in ARCHITECTURE.md)

- `main.go` stays in the repo root (keeps `go install github.com/sharon-xa/pretty-alias@latest` working). It only calls `app.Run`.
- Only `internal/app` may print to stdout/stderr, read the environment, detect the terminal, or choose an exit code (0 ok, including "no aliases"; 1 error; 2 usage).
- Nothing outside `app` calls `os.Exit`, `log.Fatal` or `panic`; return `error`.
- `shell` and `alias` know nothing about terminals or Bubble Tea. `tui` never reads files or env.
- Alias names and commands are **untrusted input**. Everything printed goes through `highlight.Sanitize` (the TUI and plain output already do). Keep it that way for any new output path.
- All loading and parsing happen **before** the TUI starts. The TUI gets valid data and has no loading or error states.
- Mouse capture stays off so users can select and copy text.

## Testing conventions

- Parser changes: add table rows in `internal/alias/parse_test.go`, and a fixture in `testdata/edge/` for anything tricky. Keep the expected row counts in `testdata/README.md` in sync.
- Every bug fix gets a regression test whose comment names the issue (e.g. `// Regression test for issue #5`).
- TUI tests drive `Update` directly with synthetic messages and assert on `m.content()` (the string inside `View()`). No teatest, no real terminal.
- Pass an explicit `colorprofile.Profile` to `highlight.New` in tests. Never use `Detect(os.Stdout, …)`: stdout isn't a terminal under `go test` or in CI.
- `testdata/` is ignored by `./...`. Don't hand-edit `testdata/large/`; regenerate it with `just fixtures`.

## Gotchas

- **GoReleaser** is pinned (v2.20.1) in the justfile and both workflows. Bump all three together.
- **Bubble Tea v2** (`charm.land/*/v2`, not `github.com/charmbracelet/*`): `View()` returns `tea.View` (alt screen is `v.AltScreen = true`). Keys are `tea.KeyPressMsg`. The space bar's key string is `"space"`, not `" "`. Viewport and help size and offset go through methods (`SetWidth`, `YOffset()`, …).
- **Colour detection**: `colorprofile.Detect` asks tmux for its real capabilities, so `TERM=...` overrides have no effect inside tmux. To test colour modes end to end, run under `env -i HOME=$HOME PATH=$PATH TERM=... COLORTERM=...`.
- **Release asset contract**: `install.sh` expects `pretty-alias-<os>-<arch>` (linux/darwin × amd64/arm64) plus `checksums.txt` in `sha256sum` format. GoReleaser makes these only because `.goreleaser.yaml` overrides its defaults (`formats: [binary]`, `checksum.name_template`). After touching `.goreleaser.yaml` or `install.sh`, run `just dist`; `scripts/check-dist.sh` checks the contract and installs the snapshot with the real installer.
- **CI is path-filtered** (`.github/workflows/ci.yml`): it runs only for Go files, `go.mod`/`go.sum`, `testdata/`, `install.sh`, `.golangci.yml`, `.goreleaser.yaml`, `scripts/` and the workflow itself, and never for `.md` files. If the build or tests start depending on another kind of file (e.g. `go:embed` assets), add it to the `paths` list.
- The debug log (`PRETTY_ALIAS_DEBUG=1`) goes to the user cache dir, never the working directory. No `.env` files.

## Code style

- Formatting comes from gofumpt + golines (100 columns) through `just fmt`. Don't hand-format against it.
- Keep comments minimal: doc comments on exported identifiers, and inline comments only where the *why* isn't obvious.
- Use American spelling (`color`) in new code and comments. Don't churn existing text just to change spelling.

## Dependencies

- Keep everything on the **latest stable** release, including breaking major versions: port the code rather than staying behind. Skip alpha/beta/rc releases.
- The `go` directive in `go.mod` tracks the latest Go release.
- **Ask before adding a new dependency.**
- CI action versions (`.github/workflows/ci.yml`) are pinned to major tags. Keep them current too.

## Docs to keep in sync

Update these in the same change as the code:

- `README.md`: flags, keys, environment variables, install, limitations (anything user-facing)
- `ARCHITECTURE.md`: packages, dependency rules, "where new features go", decisions
- `testdata/README.md`: fixture list and expected counts

## Git and GitHub

- **Never create branches** unless explicitly asked. Work and commit on the current branch (normally `main`).
- Commit when a unit of work is done and checks pass. **Ask before every push.**
- **Never create or push a tag, or create a GitHub release, without being asked.** Pushing a `v*` tag publishes a release (`.github/workflows/release.yml`).
- Use conventional commit subjects (`fix:`, `feat:`, `chore(deps):`, `ci:`, `docs:`, `test:`, `refactor:`, `build:`). When a commit completes an issue, put `Closes #N` in the body.
- If an issue is only partly done, leave a progress comment on it listing what's done and what remains, and keep it open.
- Work is tracked in GitHub issues. Roadmap and milestones are in #45. Check `gh issue list` before suggesting what to do next.
