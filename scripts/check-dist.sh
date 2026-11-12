#!/bin/sh
# Verify a GoReleaser dist/ directory against the release asset contract that install.sh
# relies on, then install from it with the real install.sh over a local HTTP server.
#
#   scripts/check-dist.sh [dist-dir]      (default: dist)
#
# Needs: python3 (for the HTTP server and reading metadata.json), curl, sha256sum or shasum.
set -eu

dist="${1:-dist}"
root="$(cd "$(dirname "$0")/.." && pwd)"
fail() {
    printf 'check-dist: %s\n' "$*" >&2
    exit 1
}
sha256() {
    if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi
}

[ -d "$dist" ] || fail "no $dist directory (run: just dist)"
dist="$(cd "$dist" && pwd)"
[ -f "$dist/artifacts.json" ] || fail "no $dist/artifacts.json (not a GoReleaser dist?)"

work="$(mktemp -d)"
server_pid=""
cleanup() {
    [ -z "$server_pid" ] || kill "$server_pid" 2>/dev/null || true
    rm -rf "$work"
}
trap cleanup EXIT

version="v$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$dist/metadata.json")"
release="$work/www/download/$version"
mkdir -p "$release" "$work/bin"

# --- 1. stage exactly what a release would upload --------------------------------------------
# artifacts.json maps each upload name (e.g. pretty-alias-linux-amd64) to its file in dist/.
# Paths in it are relative to the project root.
python3 - "$dist/artifacts.json" "$release" "$root" <<'EOF'
import json, os, shutil, sys
manifest, out, root = sys.argv[1:]
for a in json.load(open(manifest)):
    upload = a["type"] == "Checksum" or (a["type"] == "Binary" and a["name"].startswith("pretty-alias-"))
    if upload:
        shutil.copy(os.path.join(root, a["path"]), os.path.join(out, a["name"]))
EOF

# --- 2. exactly the expected assets, with matching checksums ---------------------------------
expected="pretty-alias-darwin-amd64 pretty-alias-darwin-arm64 pretty-alias-linux-amd64 pretty-alias-linux-arm64"
[ -f "$release/checksums.txt" ] || fail "missing checksums.txt"

staged="$(cd "$release" && ls | grep -v '^checksums.txt$' | sort | tr '\n' ' ' | sed 's/ $//')"
[ "$staged" = "$expected" ] || fail "release assets are [$staged], want [$expected]"
listed="$(awk '{print $2}' "$release/checksums.txt" | sort | tr '\n' ' ' | sed 's/ $//')"
[ "$listed" = "$expected" ] || fail "checksums.txt lists [$listed], want [$expected]"

for asset in $expected; do
    want="$(awk -v f="$asset" '$2 == f {print $1}' "$release/checksums.txt")"
    [ "$want" = "$(sha256 "$release/$asset")" ] || fail "checksum mismatch for $asset"
done
echo "check-dist: assets and checksums.txt match the contract"

# --- 3. install with the real install.sh from a local "release" ------------------------------
python3 - "$work/www" "$work/port" <<'EOF' &
import functools, http.server, sys
class Quiet(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *args):
        pass
srv = http.server.HTTPServer(("127.0.0.1", 0), functools.partial(Quiet, directory=sys.argv[1]))
open(sys.argv[2], "w").write(str(srv.server_port))
srv.serve_forever()
EOF
server_pid=$!

i=0
until [ -s "$work/port" ]; do
    i=$((i + 1))
    [ "$i" -lt 50 ] || fail "local HTTP server did not start"
    sleep 0.1
done

# A throwaway HOME, so the shell-completion stubs don't land in the real one.
HOME="$work/home" XDG_DATA_HOME="$work/home/.local/share" ZDOTDIR="$work/home" \
    PRETTY_ALIAS_BASE_URL="http://127.0.0.1:$(cat "$work/port")" \
    PRETTY_ALIAS_VERSION="$version" \
    INSTALL_DIR="$work/bin" \
    sh "$root/install.sh" || fail "install.sh failed against the snapshot"

out="$("$work/bin/pretty-alias" --version)"
[ "$out" = "pretty-alias $version" ] || fail "installed binary reports '$out', want 'pretty-alias $version'"
echo "check-dist: install.sh installed $version and the binary runs"
