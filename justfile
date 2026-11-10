# pretty-alias task runner. Run `just` to list recipes.

bin := "pretty-alias"
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags := "-X main.version=" + version

# list recipes
default:
    @just --list

# build the binary into ./pretty-alias (gitignored)
build:
    go build -ldflags "{{ldflags}}" -o {{bin}} .

# run the TUI against your real rc file
run:
    go run .

# run with debug logging (log goes to the user cache dir, never the CWD; see `just logs`)
debug:
    PRETTY_ALIAS_DEBUG=1 go run .

# follow the debug log (run `just debug` in another terminal)
logs:
    tail -f "$( [ "$(uname)" = Darwin ] && echo "$HOME/Library/Caches" || echo "${XDG_CACHE_HOME:-$HOME/.cache}" )/pretty-alias/debug.log"

# print the parsed aliases (no TUI) for a fixture or any file
#   just plain testdata/zshrc zsh
plain file shell="bash":
    go run . --plain --shell {{shell}} --file {{file}}

# run against a throwaway HOME built from a fixture rc file, so you can
# try edge cases without touching your real config.
#   just try testdata/bashrc            (defaults to bash)
#   just try testdata/config.fish fish
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

# (re)generate the large fixtures in testdata/large. Same seed = identical files.
#   just fixtures            (2000 aliases per shell)
#   just fixtures 20000 7    (stress test: 20k aliases, seed 7)
# NOTE: only the default (2000, seed 1) is meant to be committed.
fixtures n="2000" seed="1":
    go run testdata/gen/main.go -n {{n}} -seed {{seed}} -out testdata/large

# run every fixture through the real binary in a pty and report ok / crash / empty / hang.
# Every fixture should be "ok" or "EMPTY"; anything else is a regression.
#   just smoke               (everything)
#   just smoke edge          (only paths containing "edge")
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

# format all Go code
fmt:
    gofumpt -l -w .

# go vet
vet:
    go vet ./...

# golangci-lint
lint:
    golangci-lint run ./...

# run tests
test:
    go test ./...

# run tests with the race detector
race:
    go test -race ./...

# tests with coverage summary
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -n 1

# everything CI should run: format check, vet, lint, race tests
check: vet lint race
    @test -z "$(gofumpt -l .)" || (echo "gofumpt: unformatted files:"; gofumpt -l .; exit 1)

# tidy modules and verify
tidy:
    go mod tidy
    go mod verify

# upgrade every dependency to its latest version, then tidy and test
upgrade:
    go get -u ./...
    go mod tidy
    go test ./...

# show outdated direct dependencies
outdated:
    go list -mod=readonly -u -m -f '{{ "{{if and (not .Indirect) .Update}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}" }}' all

# install to $GOPATH/bin (or GOBIN)
install:
    go install .

# cross-compile release-style binaries into ./dist
dist:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p dist
    for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
      os=${target%/*}; arch=${target#*/}
      echo "building $os/$arch"
      GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w {{ldflags}}" \
        -o "dist/{{bin}}-$os-$arch" .
    done
    (cd dist && sha256sum {{bin}}-* > checksums.txt)

# remove build artifacts
clean:
    rm -rf {{bin}} dist coverage.out
