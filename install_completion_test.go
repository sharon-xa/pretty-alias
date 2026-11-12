package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sharon-xa/pretty-alias/v2/internal/app"
)

// Tests for the shell-completion step of install.sh.

func releaseServer(t *testing.T) installer {
	t.Helper()
	asset := platformAsset(t)
	bin := fakeBinary()
	srv := server(
		t,
		map[string][]byte{asset: bin, "checksums.txt": checksums(map[string][]byte{asset: bin})},
	)
	return installer{srv: srv, version: fakeTag, dir: filepath.Join(t.TempDir(), "bin")}
}

func mustInstall(t *testing.T, in installer) result {
	t.Helper()
	r := runInstallerWith(t, in)
	if r.err != nil {
		t.Fatalf("install failed: %v\nstdout: %s\nstderr: %s", r.err, r.out, r.errOut)
	}
	return r
}

// stubs are where install.sh puts each shell's stub for a user install.
func stubs(home string) map[string]string {
	share := filepath.Join(home, ".local", "share")
	return map[string]string{
		"bash": filepath.Join(share, "bash-completion", "completions", "pretty-alias"),
		"fish": filepath.Join(share, "fish", "vendor_completions.d", "pretty-alias.fish"),
		"zsh":  filepath.Join(share, "zsh", "site-functions", "_pretty-alias"),
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestInstallCompletionStubs(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	in.shells = []string{"bash", "zsh", "fish"}
	r := mustInstall(t, in)

	for sh, path := range stubs(in.home) {
		stub, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("no %s stub: %v\n%s", sh, err, r.out)
			continue
		}
		if !strings.Contains(string(stub), "pretty-alias --completion "+sh) {
			t.Errorf("%s stub does not load the script:\n%s", sh, stub)
		}
		if !strings.Contains(r.out, "  "+sh) {
			t.Errorf("output does not list the %s stub:\n%s", sh, r.out)
		}
	}
	if !strings.Contains(r.out, "fpath=(~/.local/share/zsh/site-functions $fpath)") ||
		!strings.Contains(r.out, "~/.zshrc") {
		t.Errorf("missing the zsh line to add:\n%s", r.out)
	}
	if files := installedFiles(in.dir); len(files) != 1 {
		t.Errorf("expected only the binary in the install dir, got %v", files)
	}
}

func TestInstallCompletionOnlyForInstalledShells(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	in.shells = []string{"fish"}
	r := mustInstall(t, in)
	for sh, path := range stubs(in.home) {
		if exists(path) != (sh == "fish") {
			t.Errorf("%s stub exists: %v, want %v", sh, exists(path), sh == "fish")
		}
	}
	if strings.Contains(r.out, "zshrc") {
		t.Errorf("zsh advice without zsh:\n%s", r.out)
	}
}

func TestInstallNoShellsNoCompletionOutput(t *testing.T) {
	in := releaseServer(t)
	r := mustInstall(t, in)
	if strings.Contains(r.out, "completion") {
		t.Errorf("completion output with no shells installed:\n%s", r.out)
	}
}

func TestInstallZshUsesWritableFpathDir(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	site := filepath.Join(t.TempDir(), "share", "zsh", "site-functions")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatal(err)
	}
	in.shells = []string{"zsh"}
	in.fpath = []string{
		filepath.Join(t.TempDir(), "missing", "site-functions"),
		filepath.Join(t.TempDir(), "functions"),
		site,
	}
	r := mustInstall(t, in)
	if !exists(filepath.Join(site, "_pretty-alias")) {
		t.Fatalf("zsh stub not in the writable fpath dir:\n%s", r.out)
	}
	if exists(stubs(in.home)["zsh"]) || strings.Contains(r.out, "fpath=(") {
		t.Errorf("a dir already in $fpath needs no setup:\n%s", r.out)
	}
}

func TestInstallZshNoteSkippedWhenZshrcHasTheDir(t *testing.T) {
	for _, line := range []string{
		"fpath=(~/.local/share/zsh/site-functions $fpath)",
		"fpath=($HOME/.local/share/zsh/site-functions $fpath)",
	} {
		in := releaseServer(t)
		in.home = t.TempDir()
		in.shells = []string{"zsh"}
		zshrc := "# my zshrc\n" + line + "\nautoload -Uz compinit && compinit\n"
		if err := os.WriteFile(filepath.Join(in.home, ".zshrc"), []byte(zshrc), 0o600); err != nil {
			t.Fatal(err)
		}
		r := mustInstall(t, in)
		if !exists(stubs(in.home)["zsh"]) {
			t.Errorf("no zsh stub:\n%s", r.out)
		}
		if strings.Contains(r.out, "fpath=(") {
			t.Errorf("asked to add a line ~/.zshrc already has (%s):\n%s", line, r.out)
		}
	}
}

func TestInstallCompletionRespectsXDGDataHome(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	data := filepath.Join(t.TempDir(), "data")
	in.shells = []string{"fish"}
	in.env = []string{"XDG_DATA_HOME=" + data}
	mustInstall(t, in)
	if !exists(filepath.Join(data, "fish", "vendor_completions.d", "pretty-alias.fish")) {
		t.Error("fish stub not under $XDG_DATA_HOME")
	}
}

func TestInstallCompletionOptOut(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	in.shells = []string{"bash", "zsh", "fish"}
	in.env = []string{"PRETTY_ALIAS_NO_COMPLETIONS=1"}
	r := mustInstall(t, in)
	for sh, path := range stubs(in.home) {
		if exists(path) {
			t.Errorf("%s stub written despite PRETTY_ALIAS_NO_COMPLETIONS", sh)
		}
	}
	if strings.Contains(r.out, "completion") {
		t.Errorf("completion output despite opting out:\n%s", r.out)
	}
}

// Releases before v2.1.0 have no --completion flag. A stub would make the shell
// run the old binary (which takes no flags) whenever the user presses tab.
func TestInstallCompletionOnlyForVersionsWithTheFlag(t *testing.T) {
	tests := map[string]bool{
		"v1.2.1": false, "v2.0.0": false, "v2.0.9": false, "2.0.1": false,
		"v2.1.0": true, "2.1.0": true, "v2.10.3": true, "v3.0.0": true, "v2.1.0-rc.1": true,
	}
	for version, want := range tests {
		t.Run(version, func(t *testing.T) {
			in := releaseServer(t)
			in.version = version
			in.home = t.TempDir()
			in.shells = []string{"fish"}
			mustInstall(t, in)
			if got := exists(stubs(in.home)["fish"]); got != want {
				t.Errorf("stub installed: %v, want %v", got, want)
			}
		})
	}
}

// TestInstalledStubsComplete loads the installed bash and fish stubs into the
// real shells, with a stand-in pretty-alias that prints the real scripts.
func TestInstalledStubsComplete(t *testing.T) {
	in := releaseServer(t)
	in.home = t.TempDir()
	in.shells = []string{"bash", "fish"}
	mustInstall(t, in)

	bin := t.TempDir()
	for _, sh := range []string{"bash", "fish"} {
		var script bytes.Buffer
		if code := app.Run([]string{"--completion", sh}, &script, io.Discard, "test"); code != 0 {
			t.Fatalf("--completion %s: exit %d", sh, code)
		}
		if err := os.WriteFile(
			filepath.Join(bin, "script."+sh),
			script.Bytes(),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	fake := "#!/bin/sh\n[ \"$1\" = --completion ] && exec cat \"" + bin + "/script.$2\"\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "pretty-alias"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	stub := stubs(in.home)

	t.Run("bash", func(t *testing.T) {
		bash, err := exec.LookPath("bash")
		if err != nil {
			t.Skip("bash not installed")
		}
		cmd := exec.Command(bash, "--norc", "--noprofile", "-c",
			`source "$1"; COMP_WORDS=(pretty-alias --sort ""); COMP_CWORD=2; _pretty_alias; `+
				`echo "${COMPREPLY[*]}"`, "bash", stub["bash"])
		cmd.Env = append(os.Environ(), "PATH="+path)
		out, err := cmd.CombinedOutput()
		if got := strings.TrimSpace(string(out)); err != nil || got != "file name command" {
			t.Errorf("got %q (%v), want the sort orders", got, err)
		}
	})
	t.Run("fish", func(t *testing.T) {
		fish, err := exec.LookPath("fish")
		if err != nil {
			t.Skip("fish not installed")
		}
		cmd := exec.Command(fish, "--no-config", "-c",
			`source $argv[1]; complete -C 'pretty-alias --sort ' | string replace -r '\t.*' ''`,
			stub["fish"])
		cmd.Env = append(os.Environ(), "PATH="+path)
		out, _ := cmd.CombinedOutput()
		if got := strings.Fields(string(out)); strings.Join(got, " ") != "command file name" {
			t.Errorf("got %q, want the sort orders", out)
		}
	})
}
