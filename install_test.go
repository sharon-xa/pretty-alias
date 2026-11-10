package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests run the real install.sh against a local fake "releases" server.
// They need sh and curl; they are skipped where those are missing.

const fakeTag = "v9.9.9"

// fakeBinary is large enough and starts with the ELF magic, so it passes the
// installer's "is this an executable" check on any platform.
func fakeBinary() []byte {
	return append([]byte{0x7f, 'E', 'L', 'F'}, bytes.Repeat([]byte("pretty-alias"), 20000)...)
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// platformAsset is the asset name install.sh looks for on this machine.
func platformAsset(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		t.Skipf("install.sh does not support %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		t.Skipf("install.sh does not support %s", runtime.GOARCH)
	}
	return "pretty-alias-" + runtime.GOOS + "-" + runtime.GOARCH
}

// server serves /download/<tag>/<name> from files; everything else is a 404.
func server(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/download/" + fakeTag + "/"
		if name, ok := strings.CutPrefix(r.URL.Path, prefix); ok {
			if data, found := files[name]; found {
				_, _ = w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

type result struct {
	out, errOut string
	err         error
	dir         string // install dir
}

func runInstaller(t *testing.T, srv *httptest.Server, version string) result {
	t.Helper()
	for _, tool := range []string{"sh", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	dir := filepath.Join(t.TempDir(), "nested", "bin") // does not exist yet
	cmd := exec.Command("sh", "install.sh")
	cmd.Env = append(os.Environ(),
		"PRETTY_ALIAS_BASE_URL="+srv.URL,
		"PRETTY_ALIAS_VERSION="+version,
		"INSTALL_DIR="+dir,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return result{out: stdout.String(), errOut: stderr.String(), err: err, dir: dir}
}

func checksums(entries map[string][]byte) []byte {
	var b strings.Builder
	for name, data := range entries {
		fmt.Fprintf(&b, "%s  %s\n", sum(data), name)
	}
	return []byte(b.String())
}

// installedFiles lists what is in the install dir (it may not exist).
func installedFiles(dir string) []string {
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func wantFailure(t *testing.T, r result, msg string) {
	t.Helper()
	if r.err == nil {
		t.Fatalf("installer succeeded, expected failure\nstdout: %s\nstderr: %s", r.out, r.errOut)
	}
	if !strings.Contains(r.errOut, msg) {
		t.Errorf("stderr does not contain %q:\n%s", msg, r.errOut)
	}
	if files := installedFiles(r.dir); len(files) != 0 {
		t.Errorf("a failed install left files behind: %v", files)
	}
}

func TestInstallSyntax(t *testing.T) {
	if out, err := exec.Command("sh", "-n", "install.sh").CombinedOutput(); err != nil {
		t.Fatalf("sh -n: %v\n%s", err, out)
	}
	script, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(script), "#!/bin/sh") || !strings.Contains(string(script), "set -eu") {
		t.Error("install.sh must be POSIX sh with `set -eu`")
	}
	if strings.Contains(string(script), "curl -Lo") || strings.Contains(string(script), "curl -L ") {
		t.Error("curl must always be called with -f (the original bug)")
	}
}

func TestInstallVerifiedRelease(t *testing.T) {
	asset := platformAsset(t)
	bin := fakeBinary()
	srv := server(t, map[string][]byte{
		asset:           bin,
		"checksums.txt": checksums(map[string][]byte{asset: bin, "pretty-alias-other-os": []byte("x")}),
	})

	r := runInstaller(t, srv, fakeTag)
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.errOut)
	}
	got, err := os.ReadFile(filepath.Join(r.dir, "pretty-alias"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bin) {
		t.Error("installed binary differs from the release asset")
	}
	if info, _ := os.Stat(filepath.Join(r.dir, "pretty-alias")); info.Mode().Perm()&0o111 == 0 {
		t.Errorf("binary is not executable: %v", info.Mode())
	}
	if !strings.Contains(r.out, "Checksum verified") {
		t.Errorf("expected a verification message:\n%s", r.out)
	}
	if files := installedFiles(r.dir); len(files) != 1 {
		t.Errorf("expected exactly the binary in the install dir, got %v", files)
	}
}

func TestInstallAcceptsVersionWithoutV(t *testing.T) {
	asset := platformAsset(t)
	bin := fakeBinary()
	srv := server(t, map[string][]byte{asset: bin, "checksums.txt": checksums(map[string][]byte{asset: bin})})
	if r := runInstaller(t, srv, strings.TrimPrefix(fakeTag, "v")); r.err != nil {
		t.Fatalf("install of bare version failed: %v\n%s", r.err, r.errOut)
	}
}

// The bug from issue #18: with `curl -Lo`, a 404 saved the error page as the
// binary and the script printed "installed successfully!".
func TestInstallFailsOn404AndInstallsNothing(t *testing.T) {
	platformAsset(t)
	r := runInstaller(t, server(t, nil), fakeTag)
	wantFailure(t, r, "no binary")
	if strings.Contains(r.out, "Installed") {
		t.Errorf("claimed success:\n%s", r.out)
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	asset := platformAsset(t)
	bin := fakeBinary()
	srv := server(t, map[string][]byte{
		asset:           bin,
		"checksums.txt": []byte(strings.Repeat("0", 64) + "  " + asset + "\n"),
	})
	wantFailure(t, runInstaller(t, srv, fakeTag), "checksum mismatch")
}

func TestInstallRefusesNewStyleReleaseWithoutChecksums(t *testing.T) {
	asset := platformAsset(t)
	srv := server(t, map[string][]byte{asset: fakeBinary()})
	wantFailure(t, runInstaller(t, srv, fakeTag), "no checksums.txt")
}

func TestInstallRejectsMissingChecksumEntry(t *testing.T) {
	asset := platformAsset(t)
	srv := server(t, map[string][]byte{
		asset:           fakeBinary(),
		"checksums.txt": []byte(sum([]byte("x")) + "  something-else\n"),
	})
	wantFailure(t, runInstaller(t, srv, fakeTag), "no entry for")
}

// A correct checksum of an HTML error page must still be rejected.
func TestInstallRejectsNonExecutable(t *testing.T) {
	asset := platformAsset(t)
	page := []byte("<!DOCTYPE html><html><body>Not Found</body></html>" + strings.Repeat(" ", 200000))
	srv := server(t, map[string][]byte{
		asset:           page,
		"checksums.txt": checksums(map[string][]byte{asset: page}),
	})
	wantFailure(t, runInstaller(t, srv, fakeTag), "not a valid executable")
}

func TestInstallRejectsTinyFile(t *testing.T) {
	asset := platformAsset(t)
	tiny := []byte{0x7f, 'E', 'L', 'F', 1, 2, 3}
	srv := server(t, map[string][]byte{asset: tiny, "checksums.txt": checksums(map[string][]byte{asset: tiny})})
	wantFailure(t, runInstaller(t, srv, fakeTag), "not a valid executable")
}

// Releases up to v1.2.1 ship one unverifiable Linux amd64 binary named
// plainly "pretty-alias". It must keep working, with a loud warning.
func TestInstallLegacyReleaseOnLinuxAmd64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("legacy asset is Linux amd64 only")
	}
	bin := fakeBinary()
	srv := server(t, map[string][]byte{"pretty-alias": bin})
	r := runInstaller(t, srv, fakeTag)
	if r.err != nil {
		t.Fatalf("legacy install failed: %v\n%s", r.err, r.errOut)
	}
	if !strings.Contains(r.errOut, "could not be verified") {
		t.Errorf("expected an unverified warning, got stderr:\n%s", r.errOut)
	}
	if got, _ := os.ReadFile(filepath.Join(r.dir, "pretty-alias")); !bytes.Equal(got, bin) {
		t.Error("installed binary differs from the legacy asset")
	}
}

func TestInstallLegacyReleaseOtherPlatformsFailClearly(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("legacy asset is valid on this platform")
	}
	platformAsset(t)
	srv := server(t, map[string][]byte{"pretty-alias": fakeBinary()})
	wantFailure(t, runInstaller(t, srv, fakeTag), "no binary for "+runtime.GOOS+"/"+runtime.GOARCH)
}

func TestInstallReplacesExistingBinaryAtomically(t *testing.T) {
	asset := platformAsset(t)
	bin := fakeBinary()
	srv := server(t, map[string][]byte{asset: bin, "checksums.txt": checksums(map[string][]byte{asset: bin})})

	r := runInstaller(t, srv, fakeTag)
	if r.err != nil {
		t.Fatal(r.errOut)
	}
	// Second run into the same directory must replace, not fail or leave temp files.
	cmd := exec.Command("sh", "install.sh")
	cmd.Env = append(os.Environ(), "PRETTY_ALIAS_BASE_URL="+srv.URL, "PRETTY_ALIAS_VERSION="+fakeTag, "INSTALL_DIR="+r.dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("re-install failed: %v\n%s", err, out)
	}
	if files := installedFiles(r.dir); len(files) != 1 || files[0] != "pretty-alias" {
		t.Errorf("unexpected files after re-install: %v", files)
	}
}

func TestInstallRejectsBadVersion(t *testing.T) {
	platformAsset(t)
	wantFailure(t, runInstaller(t, server(t, nil), "latest-and-greatest"), "invalid version")
}
