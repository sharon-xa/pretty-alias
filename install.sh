#!/bin/sh
# pretty-alias installer.
#
#   curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh | sh
#
# Environment (all optional):
#   PRETTY_ALIAS_VERSION   release to install, e.g. v1.2.1 or 1.2.1 (default: the latest release)
#   INSTALL_DIR            where to put the binary. Default: /usr/local/bin when running as
#                          root or when it is writable, otherwise ~/.local/bin
#   PRETTY_ALIAS_BASE_URL  releases URL, for mirrors and tests
#
# What it does: detects your OS and CPU, downloads the matching release asset to a temporary
# directory, verifies it against the release's checksums.txt, checks that it really is an
# executable (not an HTML error page), and only then installs it. It never writes a partial or
# unverified file to the install directory, and it never runs the downloaded binary.

set -eu

BIN="pretty-alias"
BASE_URL="${PRETTY_ALIAS_BASE_URL:-https://github.com/sharon-xa/pretty-alias/releases}"
GO_HINT="or build from source with: go install github.com/sharon-xa/pretty-alias@latest"

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
