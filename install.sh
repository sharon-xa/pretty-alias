#!/bin/sh
# pretty-alias installer.
#
#   curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh | sh
#
# Environment (all optional):
#   PRETTY_ALIAS_VERSION   release to install, e.g. v1.2.1 or 1.2.1 (default: the latest release)
#   INSTALL_DIR            where to put the binary. Default: /usr/local/bin when running as
#                          root or when it is writable, otherwise ~/.local/bin
#   PRETTY_ALIAS_NO_COMPLETIONS  set to anything to skip installing shell completion
#   PRETTY_ALIAS_BASE_URL  releases URL, for mirrors and tests
#
# What it does: detects your OS and CPU, downloads the matching release asset to a temporary
# directory, verifies it against the release's checksums.txt, checks that it really is an
# executable (not an HTML error page), and only then installs it. It never writes a partial or
# unverified file to the install directory, and it never runs the downloaded binary.
# Then, for each of bash, zsh and fish that is installed, it writes a small completion stub where
# that shell finds completions by itself (see "shell completion" below).

set -eu

BIN="pretty-alias"
BASE_URL="${PRETTY_ALIAS_BASE_URL:-https://github.com/sharon-xa/pretty-alias/releases}"
GO_HINT="or build from source with: go install github.com/sharon-xa/pretty-alias/v2@latest"

say() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
err() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

for tool in curl uname od head wc mktemp awk; do
    command -v "$tool" >/dev/null 2>&1 || err "required tool not found: $tool"
done

# --- platform -----------------------------------------------------------------------------
case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) err "unsupported OS: $(uname -s) ($GO_HINT)" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) err "unsupported CPU: $(uname -m) ($GO_HINT)" ;;
esac

# --- helpers ------------------------------------------------------------------------------
# download URL DEST: fail on HTTP errors (-f), follow redirects, retry transient failures.
download() {
    curl -fsSL --retry 3 --connect-timeout 15 -o "$2" "$1"
}

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$1" | awk '{print $NF}'
    else
        err "cannot verify checksum: none of sha256sum, shasum or openssl is installed"
    fi
}

# looks_like_binary FILE: size sanity check plus ELF / Mach-O magic number.
looks_like_binary() {
    [ "$(wc -c <"$1")" -gt 100000 ] || return 1
    case "$(head -c 4 "$1" | od -An -tx1 | tr -d ' \n')" in
    7f454c46 | cffaedfe | cefaedfe | feedfacf | feedface | cafebabe) return 0 ;;
    *) return 1 ;;
    esac
}

# --- version ------------------------------------------------------------------------------
version="${PRETTY_ALIAS_VERSION:-}"
if [ -z "$version" ]; then
    # /releases/latest redirects to /releases/tag/<tag>; no API token or rate limit involved.
    final_url="$(curl -fsSLI --retry 3 --connect-timeout 15 -o /dev/null -w '%{url_effective}' "$BASE_URL/latest")" ||
        err "could not look up the latest release at $BASE_URL/latest (set PRETTY_ALIAS_VERSION to skip this)"
    version="${final_url##*/}"
fi
case "$version" in
v[0-9]*) ;;
[0-9]*) version="v$version" ;;
*) err "invalid version: '$version' (expected something like v1.2.1)" ;;
esac

# --- install directory --------------------------------------------------------------------
if [ -n "${INSTALL_DIR:-}" ]; then
    dir="$INSTALL_DIR"
elif [ "$(id -u)" = 0 ] || [ -w /usr/local/bin ]; then
    dir=/usr/local/bin
else
    dir="${HOME:?HOME is not set}/.local/bin"
fi
mkdir -p "$dir" 2>/dev/null || err "cannot create $dir (re-run with sudo, or set INSTALL_DIR)"
[ -w "$dir" ] || err "cannot write to $dir (re-run with sudo, or set INSTALL_DIR)"

# --- download -----------------------------------------------------------------------------
tmp="$(mktemp -d)"
staged=""
cleanup() {
    rm -rf "$tmp"
    [ -z "$staged" ] || rm -f "$staged"
}
trap cleanup EXIT
trap 'exit 130' INT TERM HUP

release_url="$BASE_URL/download/$version"
legacy=0
asset="$BIN-$os-$arch"
say "Installing $BIN $version for $os/$arch..."

if ! download "$release_url/$asset" "$tmp/$asset" 2>/dev/null; then
    # Releases up to v1.2.1 published a single Linux amd64 binary named plainly "pretty-alias".
    if [ "$os-$arch" = "linux-amd64" ]; then
        legacy=1
        asset="$BIN"
        download "$release_url/$asset" "$tmp/$asset" 2>/dev/null ||
            err "download failed: no binary found for $version at $release_url ($GO_HINT)"
    else
        err "release $version has no binary for $os/$arch at $release_url/$asset ($GO_HINT)"
    fi
fi

# --- verify -------------------------------------------------------------------------------
if download "$release_url/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
    expected="$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1; exit }' "$tmp/checksums.txt")"
    [ -n "$expected" ] || err "checksums.txt for $version has no entry for $asset"
    actual="$(sha256_of "$tmp/$asset")"
    [ "$expected" = "$actual" ] || err "checksum mismatch for $asset (expected $expected, got $actual); refusing to install"
    say "Checksum verified."
elif [ "$legacy" = 1 ]; then
    warn "$version predates published checksums, so the download could not be verified."
else
    err "$version has no checksums.txt; refusing to install an unverified binary"
fi

looks_like_binary "$tmp/$asset" ||
    err "the downloaded file is not a valid executable (wrong URL or an error page?); nothing was installed"

# --- install ------------------------------------------------------------------------------
# Stage next to the destination, then rename: the move is atomic, so an interrupted install
# never leaves a half-written $BIN behind or breaks an existing one.
staged="$dir/.$BIN.$$"
cp "$tmp/$asset" "$staged"
chmod 755 "$staged"
mv -f "$staged" "$dir/$BIN"
staged=""

say "Installed $dir/$BIN"
case ":${PATH:-}:" in
*":$dir:"*) ;;
*) say "Note: $dir is not in your PATH. Add it, for example: export PATH=\"$dir:\$PATH\"" ;;
esac

# --- shell completion ---------------------------------------------------------------------
# For each installed shell, write a small stub where that shell loads completions by itself.
# The stub asks pretty-alias for the script when the shell first needs it, so it stays current
# after upgrades, and this installer still never runs the binary. Nothing here can fail the
# install.

# supports_completion: --completion exists from v2.1.0 on. Older binaries take no flags and
# would start the TUI from inside the shell's completion code, so they get no stub.
supports_completion() {
    rest="${version#v}"
    major="${rest%%.*}"
    rest="${rest#"$major"}"
    rest="${rest#.}"
    minor="${rest%%[!0-9]*}"
    case "$major" in '' | *[!0-9]*) return 1 ;; esac
    [ -n "$minor" ] || minor=0
    [ "$major" -gt 2 ] || { [ "$major" -eq 2 ] && [ "$minor" -ge 1 ]; }
}

# put_file PATH CONTENT: write atomically, creating the directory.
put_file() {
    mkdir -p "$(dirname "$1")" 2>/dev/null &&
        printf '%s\n' "$2" >"$1.$$" &&
        mv -f "$1.$$" "$1"
}

# tilde PATH: show a path under $HOME as ~/...
tilde() {
    [ -n "${HOME:-}" ] || {
        printf '%s' "$1"
        return
    }
    case "$1" in
    "$HOME"/*) printf '~/%s' "${1#"$HOME"/}" ;;
    *) printf '%s' "$1" ;;
    esac
}

# zsh_site_dir: a site-functions directory in zsh's default $fpath that we can write to
# (e.g. /usr/local/share/zsh/site-functions as root, or with Homebrew on Intel Macs).
zsh_site_dir() {
    zsh -fc 'print -rl -- $fpath' 2>/dev/null | while IFS= read -r d; do
        case "$d" in
        */site-functions)
            if { [ -d "$d" ] && [ -w "$d" ]; } || { [ "$root" = 1 ] && mkdir -p "$d" 2>/dev/null; }; then
                printf '%s\n' "$d"
                break
            fi
            ;;
        esac
    done
}

STUB_NOTE="# pretty-alias completion, written by its installer. It loads the script from
# pretty-alias itself, so it stays current when pretty-alias is upgraded."

install_completions() {
    root=0
    [ "$(id -u)" = 0 ] && root=1
    [ "$root" = 1 ] || [ -n "${HOME:-}" ] || return 0
    data_home="${XDG_DATA_HOME:-$HOME/.local/share}"
    [ "$root" = 0 ] || data_home=/usr/local/share
    installed=""
    zsh_note=""
    bash_note=""

    if command -v bash >/dev/null 2>&1; then
        f="$data_home/bash-completion/completions/$BIN"
        if put_file "$f" "$STUB_NOTE
if command -v $BIN >/dev/null 2>&1; then
    eval \"\$($BIN --completion bash 2>/dev/null)\"
fi"; then
            installed="$installed
  bash  $(tilde "$f")"
            bash_note=1
            for c in /usr/share/bash-completion/bash_completion /etc/bash_completion \
                /usr/local/share/bash-completion/bash_completion \
                /usr/local/etc/profile.d/bash_completion.sh \
                /opt/homebrew/etc/profile.d/bash_completion.sh; do
                [ -f "$c" ] && bash_note="" && break
            done
        fi
    fi

    if command -v fish >/dev/null 2>&1; then
        f="$data_home/fish/vendor_completions.d/$BIN.fish"
        if put_file "$f" "$STUB_NOTE
command -q $BIN; and $BIN --completion fish 2>/dev/null | source"; then
            installed="$installed
  fish  $(tilde "$f")"
        fi
    fi

    if command -v zsh >/dev/null 2>&1; then
        zdir="$(zsh_site_dir)"
        if [ -z "$zdir" ] && [ "$root" = 0 ]; then
            zdir="$data_home/zsh/site-functions"
            zsh_note=1
            for form in "$zdir" "$(tilde "$zdir")" "\$HOME${zdir#"$HOME"}"; do
                grep -qF "$form" "${ZDOTDIR:-$HOME}/.zshrc" 2>/dev/null && zsh_note="" && break
            done
        fi
        f="$zdir/_$BIN"
        # Autoloaded as the _pretty-alias function: it replaces itself with the real one.
        if [ -n "$zdir" ] && put_file "$f" "#compdef $BIN
$STUB_NOTE
local script
script=\"\$($BIN --completion zsh 2>/dev/null)\" || return 1
[[ \$script == *'_$BIN()'* ]] || return 1
eval \"\$script\"
_$BIN \"\$@\""; then
            installed="$installed
  zsh   $(tilde "$f")"
        else
            zsh_note=""
        fi
    fi

    [ -n "$installed" ] || return 0
    say ""
    say "Installed shell completion (open a new shell to use it):$installed"
    if [ -n "$bash_note" ]; then
        say ""
        say "bash completion needs the bash-completion package, which wasn't found. Install it"
        say "with your package manager, or add this line to your ~/.bashrc instead:"
        say ""
        say "    eval \"\$($BIN --completion bash)\""
    fi
    if [ -n "$zsh_note" ]; then
        say ""
        say "To turn on zsh completion, add this line to your ~/.zshrc, above the line that"
        say "runs compinit (with oh-my-zsh: above 'source \$ZSH/oh-my-zsh.sh'):"
        say ""
        say "    fpath=($(tilde "$zdir") \$fpath)"
    fi
}

if [ -z "${PRETTY_ALIAS_NO_COMPLETIONS:-}" ] && supports_completion; then
    install_completions || warn "could not install shell completion (pretty-alias itself is installed)"
fi
