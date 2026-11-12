# pretty-alias task runner. Run `just` to list recipes.

bin := "pretty-alias"
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags := "-X main.version=" + version

# GoReleaser, pinned and run through `go run` so nothing has to be installed
goreleaser := "go run github.com/goreleaser/goreleaser/v2@v2.20.1"

# list recipes, grouped
[private]
default:
    @just --list --unsorted

# run the TUI against your real rc file
[group('run')]
run:
    go run .

# run against a throwaway HOME built from a fixture rc file, so you can
# try edge cases without touching your real config.
#   just try testdata/bashrc            (defaults to bash)
#   just try testdata/config.fish fish
[group('run')]
[doc('open the TUI on a fixture rc file in a throwaway HOME')]
try rc shell="bash":
    #!/usr/bin/env bash
    set -euo pipefail
    home=$(mktemp -d)
    trap 'rm -rf "$home"' EXIT
    case "{{shell}}" in
      bash) dest="$home/.bashrc" ;;
      zsh)  dest="$home/.zshrc" ;;
      fish) mkdir -p "$home/.config/fish"; dest="$home/.config/fish/config.fish" ;;
      *)    echo "unsupported shell: {{shell}}" >&2; exit 2 ;;
    esac
    cp "{{rc}}" "$dest"
    go build -o "$home/pa" .
    HOME="$home" SHELL="$(command -v {{shell}} || echo /usr/bin/{{shell}})" "$home/pa"

# print the parsed aliases (no TUI) for a fixture or any file
#   just plain testdata/zshrc zsh
[group('run')]
[doc('print the parsed aliases (no TUI) for a fixture or any file')]
plain file shell="bash":
    go run . --plain --shell {{shell}} --file {{file}}

# run with debug logging (log goes to the user cache dir, never the CWD; see `just logs`)
[group('run')]
debug:
    PRETTY_ALIAS_DEBUG=1 go run .

# follow the debug log (run `just debug` in another terminal)
[group('run')]
logs:
    tail -f "$( [ "$(uname)" = Darwin ] && echo "$HOME/Library/Caches" || echo "${XDG_CACHE_HOME:-$HOME/.cache}" )/pretty-alias/debug.log"

[group('test')]
[doc('what CI runs: vet, lint (incl. format check), race tests')]
check: vet lint race

# run tests
[group('test')]
test:
    go test ./...

# run tests with the race detector
[group('test')]
race:
    go test -race ./...

# tests with coverage summary
[group('test')]
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -n 1

# run every fixture through the real binary in a pty and report ok / crash / empty / hang.
# Every fixture should be "ok" or "EMPTY"; anything else is a regression.
#   just smoke               (everything)
#   just smoke edge          (only paths containing "edge")
[group('test')]
[doc('run every fixture through the real binary in a pty; all must be ok or EMPTY')]
smoke filter="":
    #!/usr/bin/env bash
    set -uo pipefail
    root="$PWD"
    work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
    go build -o "$work/pa" . || exit 1
    pass=0; fail=0
    while IFS= read -r f; do
      case "$f" in *edge/*|*empty/*|*large/*|*minimal/*|*sourced/*|testdata/bashrc|testdata/zshrc|testdata/config.fish) ;; *) continue ;; esac
      case "$f" in *.sh|*bash_aliases|*zsh_aliases|*conf.d/*) continue ;; esac   # include targets, not entry points
      [[ -n "{{filter}}" && "$f" != *"{{filter}}"* ]] && continue
      case "$f" in *zshrc*) sh=zsh ;; *fish*) sh=fish ;; *) sh=bash ;; esac
      home="$work/home"; rm -rf "$home"; mkdir -p "$home/.config/fish"
      case $sh in bash) dest=.bashrc ;; zsh) dest=.zshrc ;; fish) dest=.config/fish/config.fish ;; esac
      cp "$root/$f" "$home/$dest"
      keys=0; [[ "$f" == *large/* ]] && keys=600
      ( sleep 1; for _ in $(seq "$keys"); do printf '\033[B'; done; sleep 0.3; printf q ) \
        | timeout 20 script -qec "stty cols 120 rows 40; env HOME=$home SHELL=/usr/bin/$sh $work/pa" "$work/out" >/dev/null 2>&1
      code=$?
      if   grep -aqE 'panic|program was killed' "$work/out"; then status="CRASH (panic)"
      elif [[ $code -eq 124 ]];                     then status="HANG"
      elif grep -aq 'no aliases found' "$work/out"; then status="EMPTY (no aliases found)"
      elif [[ $code -ne 0 ]];                       then status="EXIT $code (silent)"
      else status="ok"; fi
      printf '%-48s %-6s %s\n' "$f" "$sh" "$status"
      [[ "$status" == ok* || "$status" == EMPTY* ]] && pass=$((pass+1)) || fail=$((fail+1))
    done < <(cd "$root" && find testdata -type f ! -name README.md ! -path 'testdata/gen/*' | sort)
    echo "---- ok/empty: $pass   failing: $fail"

# (re)generate the large fixtures in testdata/large. Same seed = identical files.
#   just fixtures            (2000 aliases per shell)
#   just fixtures 20000 7    (stress test: 20k aliases, seed 7)
# NOTE: only the default (2000, seed 1) is meant to be committed.
[group('test')]
[doc('regenerate testdata/large (deterministic; commit only the default n/seed)')]
fixtures n="2000" seed="1":
    go run testdata/gen/main.go -n {{n}} -seed {{seed}} -out testdata/large

# format all Go code (gofumpt + golines, configured in .golangci.yaml)
[group('lint')]
fmt:
    golangci-lint fmt ./...

# go vet
[group('lint')]
vet:
    go vet ./...

# golangci-lint
[group('lint')]
lint:
    golangci-lint run ./...

# build the binary into ./pretty-alias (gitignored)
[group('build')]
build:
    go build -ldflags "{{ldflags}}" -o {{bin}} .

# install to $GOPATH/bin (or GOBIN)
[group('build')]
install:
    go install .

# build release binaries + checksums.txt into ./dist (GoReleaser snapshot, nothing is
# published), then check them against the install.sh contract by installing from them
[group('build')]
[doc('GoReleaser snapshot into ./dist, then install it with install.sh to check it')]
dist:
    {{goreleaser}} check
    {{goreleaser}} release --snapshot --clean
    scripts/check-dist.sh dist

# remove build artifacts
[group('build')]
clean:
    rm -rf {{bin}} dist coverage.out

# tidy modules and verify
[group('deps')]
tidy:
    go mod tidy
    go mod verify

# upgrade every dependency to its latest version, then tidy and test
[group('deps')]
upgrade:
    go get -u ./...
    go mod tidy
    go test ./...

# show outdated direct dependencies
[group('deps')]
outdated:
    go list -mod=readonly -u -m -f '{{ "{{if and (not .Indirect) .Update}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}" }}' all
