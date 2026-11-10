// Command gen writes large, diverse, deterministic rc-file fixtures.
//
//	go run testdata/gen/main.go -n 2000 -seed 1 -out testdata/large
//
// It writes bashrc, zshrc and config.fish into -out. The files only contain
// aliases the current parser can handle (one per line, column 0, with a value),
// so they work as a load / scrolling / wrapping test. Known-bad inputs live in
// testdata/edge instead.
//
// The first line of every file is a machine-readable header:
//
//	# pa-fixture: shell=bash aliases=2000 seed=1
//
// so tests can learn the expected number of rows without re-counting.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

type gen struct {
	r     *rand.Rand
	names map[string]bool
}

func (g *gen) pick(s []string) string { return s[g.r.Intn(len(s))] }
func (g *gen) chance(p float64) bool  { return g.r.Float64() < p }

var (
	gitSubs    = []string{"status -sb", "log --oneline --graph --decorate", "diff --stat", "diff --staged", "commit -v", "commit --amend --no-edit", "push --force-with-lease", "pull --rebase --autostash", "fetch --all --prune", "stash push -u", "rebase -i HEAD~5", "checkout -", "switch -c", "branch -vv", "reset --soft HEAD~1", "clean -fdn", "blame -w -C", "cherry-pick -x", "bisect start", "worktree list", "submodule update --init --recursive", "log --since=yesterday --author=me --pretty=format:\"%h %s\"", "shortlog -sn --no-merges", "tag -l --sort=-v:refname"}
	dockerSubs = []string{"ps -a", "images -a", "system prune -af", "volume ls", "network ls", "logs -f --tail 100", "stats --no-stream", "compose up -d", "compose down -v", "compose logs -f", "compose ps", "compose build --no-cache", "buildx build --load -t app:dev .", "exec -it app sh", "run --rm -it alpine sh", "inspect --format '{{.State.Status}}'"}
	kubeVerbs  = []string{"get pods", "get svc", "get nodes -o wide", "get ingress", "get events --sort-by=.lastTimestamp", "describe pod", "logs -f --tail=200", "top pods", "rollout restart deploy", "port-forward svc/web 8080:80", "config current-context", "apply -f .", "delete pod --force", "exec -it deploy/web -- sh"}
	kubeNS     = []string{"", "", "-n kube-system", "-n default", "-n staging", "-n prod", "-A", "--context minikube"}
	units      = []string{"nginx", "sshd", "docker", "postgresql", "redis", "NetworkManager", "bluetooth", "cron", "tailscaled", "pipewire", "wireplumber"}
	sysVerbs   = []string{"status --no-pager", "restart", "reload", "enable --now", "disable --now", "is-active", "cat", "list-dependencies"}
	pkgCmds    = []string{"sudo apt update && sudo apt upgrade -y", "sudo apt install", "sudo apt autoremove --purge -y", "sudo pacman -Syu", "sudo pacman -Rns", "pacman -Qtdq", "yay -Sua --noconfirm", "sudo dnf upgrade --refresh", "brew update && brew upgrade", "nix-env -iA nixpkgs.", "flatpak update -y", "snap refresh", "pipx upgrade-all", "rustup update", "go install ./..."}
	lsCmds     = []string{"ls -alFh", "ls -lhtr", "ls -lhSr", "ls -d */", "eza -la --git --icons", "eza --tree --level=2", "exa -l --group-directories-first", "ls --color=auto -1", "tree -L 2 -I 'node_modules|.git'"}
	dirs       = []string{"Projects", "Downloads", "Documents", ".config", ".local/share", "work/api", "work/web", "work/infra", "src/github.com/me", "notes", "dotfiles", "Pictures/screenshots", "/var/log", "/etc/nginx", "/srv/www", "/tmp", "/opt"}
	hosts      = []string{"prod-web-01", "prod-web-02", "staging", "db1.internal", "bastion.example.invalid", "raspberrypi.local", "nas", "10.0.0.12", "build-01", "gpu-box"}
	users      = []string{"deploy", "admin", "ubuntu", "pi", "root", "me", "ci"}
	buildCmds  = []string{"go build ./...", "go test -race -count=1 ./...", "go vet ./... && staticcheck ./...", "cargo build --release", "cargo test -- --nocapture", "cargo clippy --all-targets -- -D warnings", "npm run build", "npm run dev", "pnpm install --frozen-lockfile", "pnpm test --watch", "yarn lint --fix", "make -j$(nproc)", "make clean all", "cmake -S . -B build && cmake --build build -j", "mvn -q -DskipTests package", "gradle build --info", "python3 -m pytest -xvs", "ruff check --fix . && ruff format .", "mypy --strict .", "uv sync && uv run pytest", "zig build -Doptimize=ReleaseFast", "dotnet watch run"}
	filterCmds = []string{"ps aux | grep -v grep | grep -i", "ss -tulpn | grep LISTEN", "du -ah . | sort -rh | head -n 20", "df -hT | grep -v tmpfs", "history | grep", "find . -type f -name '*.log' -mtime +7 -delete", "rg -n 'TODO|FIXME|HACK'", "fd -e md -x wc -l | sort -rn | head", "journalctl -p err -b | tail -n 50", "lsof -iTCP -sTCP:LISTEN -n -P", "awk '{print $1}' | sort | uniq -c | sort -rn", "sort | uniq -c | sort -rn", "cut -d: -f1 /etc/passwd | sort", "xargs -n1 -P4", "jq -C . | less -R", "column -t -s,"}
	netCmds    = []string{"curl -s https://ifconfig.me", "curl -sS -o /dev/null -w '%{http_code}' https://example.invalid", "ping -c 5 1.1.1.1", "dig +short", "nmap -sV -T4", "ip -br addr", "ip route", "nmcli device wifi list", "tailscale status", "mtr -rwc 10 example.invalid", "wget -c", "http --follow", "openssl s_client -connect example.invalid:443 </dev/null"}
	toolsMisc  = []string{"tmux new-session -A -s main", "tmux ls", "nvim .", "code .", "xdg-open .", "wl-copy", "xclip -selection clipboard", "bat --paging=never", "tar -xvf", "tar -czvf archive.tgz", "unzip -l", "rsync -avh --progress --partial", "gpg --list-secret-keys --keyid-format=long", "ssh-add -l", "date +%s", "date -u +%Y-%m-%dT%H:%M:%SZ", "free -h", "lscpu | grep -E 'Model name|MHz'", "sensors", "btop", "ncdu", "tldr", "python3 -m http.server 8000", "python3 -m json.tool", "openssl rand -hex 16", "sqlite3 -header -column", "psql -h localhost -U postgres", "redis-cli -h localhost", "terraform plan -out=tfplan", "aws sts get-caller-identity", "gh pr create --fill --web", "ansible-playbook -i inventory"}
	unicodeCmd = []string{"echo \"✅ done\"", "echo \"❌ failed\"", "echo \"🚀 launching\"", "echo \"日本語のテスト\"", "echo \"café naïve résumé\"", "echo \"→ next ← prev ↑ ↓\"", "echo \"★ ☆ ♥ ♦ ♣ ♠\""}
	envPrefix  = []string{"LC_ALL=C ", "TZ=UTC ", "FOO=bar ", "GOFLAGS=-mod=mod ", "RUST_LOG=debug ", "NODE_ENV=production ", "PAGER=cat ", "DEBUG=1 ", "AWS_PROFILE=prod "}
	sections   = []string{"navigation", "listing", "git", "containers", "kubernetes", "system", "packages", "network", "build & test", "text processing", "ssh", "editors", "misc", "work", "personal", "experiments", "legacy", "temporary"}
)

// command returns a random shell command without a leading alias name. It may
// contain single quotes OR double quotes but the caller decides how to wrap it.
func (g *gen) command() string {
	var c string
	switch g.r.Intn(16) {
	case 0, 1:
		c = "git " + g.pick(gitSubs)
	case 2:
		c = "docker " + g.pick(dockerSubs)
	case 3:
		c = strings.TrimSpace("kubectl " + g.pick(kubeVerbs) + " " + g.pick(kubeNS))
	case 4:
		c = "sudo systemctl " + g.pick(sysVerbs) + " " + g.pick(units)
	case 5:
		c = g.pick(pkgCmds)
	case 6:
		c = g.pick(lsCmds)
	case 7:
		c = "cd " + g.pickDir()
	case 8:
		c = "ssh " + g.pick(users) + "@" + g.pick(hosts)
		if g.chance(0.3) {
			c += " -p " + fmt.Sprint(2200+g.r.Intn(50))
		}
	case 9, 10:
		c = g.pick(buildCmds)
	case 11, 12:
		c = g.pick(filterCmds)
	case 13:
		c = g.pick(netCmds)
	case 14:
		c = g.pick(toolsMisc)
	case 15:
		if g.chance(0.5) {
			c = g.pick(unicodeCmd)
		} else {
			c = "echo \"$(date +%F) $(hostname) $USER\""
		}
	}
	if g.chance(0.04) {
		c = g.pick(envPrefix) + c
	}
	if g.chance(0.05) {
		c += " && " + g.pick([]string{"echo ok", "notify-send done", "date", "echo \"finished at $(date +%T)\""})
	}
	if g.chance(0.03) {
		c += " 2>&1 | tee /tmp/out.log"
	}
	if g.chance(0.015) { // long command: forces wrapping
		var parts []string
		for i := 0; i < 8+g.r.Intn(30); i++ {
			parts = append(parts, fmt.Sprintf("--opt%d=value%d", i, g.r.Intn(1000)))
		}
		c += " " + strings.Join(parts, " ")
	}
	return c
}

func (g *gen) pickDir() string {
	d := g.pick(dirs)
	if strings.HasPrefix(d, "/") {
		return d
	}
	return "~/" + d
}

// aliasName derives a plausible, unique alias name from the command.
func (g *gen) aliasName(cmd string) string {
	var initials []byte
	for _, w := range strings.Fields(cmd) {
		w = strings.TrimLeft(w, "-~/$(\"'")
		if w == "" || w == "sudo" || strings.Contains(w, "=") {
			continue
		}
		if c := w[0]; c >= 'a' && c <= 'z' {
			initials = append(initials, c)
		}
		if len(initials) == 3 {
			break
		}
	}
	base := string(initials)
	if base == "" {
		base = "x"
	}
	name := base
	for i := 0; g.names[name]; i++ {
		switch {
		case i < 6:
			name = base + string(rune('a'+g.r.Intn(26)))
		case i < 40:
			name = base + string(rune('a'+g.r.Intn(26))) + string(rune('a'+g.r.Intn(26)))
		default:
			name = fmt.Sprintf("%s%d", base, g.r.Intn(100000))
		}
	}
	g.names[name] = true
	return name
}

// quote wraps cmd for POSIX-ish shells. ok is false if cmd can't be represented
// simply (contains both quote kinds).
func quote(cmd string, style int) (string, bool) {
	hasSQ, hasDQ := strings.Contains(cmd, "'"), strings.Contains(cmd, "\"")
	switch {
	case hasSQ && hasDQ:
		return "", false
	case hasSQ: // must use double quotes; escape $ and ` so they stay literal
		return "\"" + strings.NewReplacer("$", "\\$", "`", "\\`").Replace(cmd) + "\"", true
	case hasDQ:
		return "'" + cmd + "'", true
	}
	switch style {
	case 0, 1, 2, 3: // mostly single quotes
		return "'" + cmd + "'", true
	case 4, 5:
		return "\"" + cmd + "\"", true
	default:
		if !strings.ContainsAny(cmd, " \t;&|<>()$`\\\"'*?[]#~{}") {
			return cmd, true // bare word
		}
		return "'" + cmd + "'", true
	}
}

func (g *gen) aliasLine(shell, cmd string) (string, bool) {
	q, ok := quote(cmd, g.r.Intn(8))
	if !ok {
		return "", false
	}
	name := g.aliasName(cmd)
	if shell == "fish" && g.chance(0.5) {
		return fmt.Sprintf("alias %s %s", name, q), true // fish: space-separated form
	}
	return fmt.Sprintf("alias %s=%s", name, q), true
}

var noise = map[string][]string{
	"bash": {
		"export HISTSIZE=50000", "export EDITOR=nvim", "shopt -s histappend", "shopt -s checkwinsize",
		"mkcd() { mkdir -p \"$1\" && cd \"$1\"; }", "[ -f ~/.fzf.bash ] && source ~/.fzf.bash",
		"export PATH=\"$HOME/.local/bin:$PATH\"", "unalias ll 2>/dev/null", "PS1='\\u@\\h:\\w\\$ '",
		"eval \"$(starship init bash)\"", "complete -cf sudo",
	},
	"zsh": {
		"setopt AUTO_CD HIST_IGNORE_DUPS SHARE_HISTORY", "autoload -Uz compinit && compinit", "bindkey -e",
		"zstyle ':completion:*' menu select", "export EDITOR=nvim", "typeset -U path",
		"eval \"$(zoxide init zsh)\"", "source $ZSH/oh-my-zsh.sh", "plugins=(git docker fzf)",
		"autoload -Uz vcs_info", "unalias ll 2>/dev/null",
	},
	"fish": {
		"set -gx EDITOR nvim", "set -gx GOPATH $HOME/go", "fish_add_path $HOME/.local/bin",
		"set fish_greeting", "abbr -a gco git checkout", "abbr -a dcu docker compose up -d",
		"starship init fish | source", "zoxide init fish | source", "functions -q fisher; or echo no fisher",
	},
}

func (g *gen) write(path, shell string, n int, seed int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	fmt.Fprintf(w, "# pa-fixture: shell=%s aliases=%d seed=%d\n", shell, n, seed)
	fmt.Fprintf(w, "# generated by testdata/gen/main.go - do not edit by hand (run `just fixtures`)\n\n")

	g.names = map[string]bool{}
	written, section := 0, 0
	for written < n {
		section++
		fmt.Fprintf(w, "# %s %s %s\n", strings.Repeat("-", 20), g.pick(sections), strings.Repeat("-", 20))
		block := 5 + g.r.Intn(40)
		for i := 0; i < block && written < n; i++ {
			line, ok := g.aliasLine(shell, g.command())
			if !ok {
				continue
			}
			fmt.Fprintln(w, line)
			written++
			switch {
			case g.chance(0.04):
				fmt.Fprintf(w, "# %s\n", g.pick([]string{"TODO: revisit", "works on my machine", "see wiki", "temporary", "legacy", "do not remove"}))
			case g.chance(0.03):
				fmt.Fprintln(w)
			case g.chance(0.03):
				fmt.Fprintln(w, g.pick(noise[shell]))
			}
		}
		fmt.Fprintln(w)
		if g.chance(0.5) {
			fmt.Fprintln(w, g.pick(noise[shell]))
		}
	}
	return nil
}

func main() {
	n := flag.Int("n", 2000, "aliases per file")
	seed := flag.Int64("seed", 1, "random seed (same seed = identical output)")
	out := flag.String("out", "testdata/large", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	files := []struct{ file, shell string }{{"bashrc", "bash"}, {"zshrc", "zsh"}, {"config.fish", "fish"}}
	for i, f := range files {
		g := &gen{r: rand.New(rand.NewSource(*seed + int64(i)))}
		p := filepath.Join(*out, f.file)
		if err := g.write(p, f.shell, *n, *seed); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d aliases)\n", p, *n)
	}
}
