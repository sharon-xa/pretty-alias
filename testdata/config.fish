# ~/.config/fish/config.fish  (pretty-alias fixture: realistic, diverse, valid for the current parser)
set -gx EDITOR nvim
set -gx VISUAL nvim
set -gx PAGER less
set -gx LESS "-R --use-color"
set -gx GOPATH $HOME/go
set -gx FZF_DEFAULT_COMMAND "fd --type f --hidden --exclude .git"
fish_add_path $HOME/.local/bin $HOME/.cargo/bin $GOPATH/bin

set fish_greeting

function mkcd
    mkdir -p $argv[1]; and cd $argv[1]
end

function fish_user_key_bindings
    fish_default_key_bindings
    bind \cf forward-char
end

# ---------------------------------------------------------------- navigation
alias .. 'cd ..'
alias ... 'cd ../..'
alias .... 'cd ../../..'
alias ~ 'cd ~'
alias cdp 'cd ~/Projects'
alias cdc 'cd ~/.config'
alias cdf 'cd ~/.config/fish'
alias cdd='cd ~/Downloads'
alias cdt='cd /tmp'

# ---------------------------------------------------------------- listing
alias ls 'eza --icons --group-directories-first'
alias ll 'eza -lah --icons --git'
alias la 'eza -a --icons'
alias lt 'eza --tree --level=2 --icons'
alias l='ls -CF'
alias cat 'bat --style=plain'
alias grep 'grep --color=auto'
alias tree='tree -C'

# ---------------------------------------------------------------- git
alias g git
alias gs 'git status -sb'
alias ga 'git add'
alias gaa 'git add --all'
alias gc 'git commit -v'
alias gcm 'git commit -m'
alias gca 'git commit --amend --no-edit'
alias gco 'git checkout'
alias gcb 'git checkout -b'
alias gd 'git diff'
alias gds 'git diff --staged'
alias gl 'git log --oneline --graph --decorate -n 20'
alias gp 'git push'
alias gpf 'git push --force-with-lease'
alias gpl 'git pull --rebase --autostash'
alias gst 'git stash push -u'
alias gstp 'git stash pop'
alias gundo 'git reset --soft HEAD~1'
alias groot 'cd (git rev-parse --show-toplevel)'
alias glp='git log --pretty=format:"%h %s (%cr) <%an>"'
alias gbr='git for-each-ref --sort=-committerdate refs/heads --format="%(refname:short) %(committerdate:relative)"'
alias lg lazygit
alias ghpr 'gh pr create --fill --web'

# ---------------------------------------------------------------- containers / cloud
alias d docker
alias dps 'docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"'
alias dex 'docker exec -it'
alias dc 'docker compose'
alias dcu 'docker compose up -d'
alias dcd 'docker compose down'
alias dcl 'docker compose logs -f --tail=100'
alias dprune 'docker system prune -af --volumes'
alias k kubectl
alias kgp 'kubectl get pods -o wide'
alias kl 'kubectl logs -f --tail=200'
alias kctx 'kubectl config current-context'
alias tf terraform
alias tfp 'terraform plan -out=tfplan'
alias awswho='aws sts get-caller-identity --output table'

# ---------------------------------------------------------------- system
alias sc 'sudo systemctl'
alias scu 'systemctl --user'
alias jc 'journalctl -xe'
alias please 'sudo $history[1]'
alias reload 'source ~/.config/fish/config.fish; and echo reloaded'
alias now 'date +"%Y-%m-%d %H:%M:%S"'
alias ports 'ss -tulpn'
alias myip 'curl -s https://ifconfig.me; echo'
alias psg 'ps aux | grep -v grep | grep -i'
alias top10 'ps aux --sort=-%mem | head -n 11'
alias cls clear
alias dfh='df -hT | grep -v tmpfs'
alias dud='du -h --max-depth=1 | sort -hr'
alias pacu='sudo pacman -Syu'
alias pacs='pacman -Ss'
alias pacr='sudo pacman -Rns'
alias yayu='yay -Sua --noconfirm'
alias orphans='pacman -Qtdq | sudo pacman -Rns -'

# ---------------------------------------------------------------- dev
alias vim nvim
alias vi nvim
alias v 'nvim .'
alias got 'go test -race -count=1 ./...'
alias gob 'go build ./...'
alias cb 'cargo build'
alias cr 'cargo run --release'
alias ct 'cargo test -- --nocapture'
alias pn pnpm
alias pnd 'pnpm dev'
alias py python3
alias venv 'python3 -m venv .venv; and source .venv/bin/activate.fish'
alias serve 'python3 -m http.server 8000'
alias json 'python3 -m json.tool'
alias m make
alias mj 'make -j(nproc)'
alias todo 'rg -n "TODO|FIXME|HACK|XXX"'
alias findf='find . -type f -name'

# ---------------------------------------------------------------- misc
alias weather='curl -s "wttr.in/?format=3"'
alias copy='wl-copy'
alias paste='wl-paste'
alias tm 'tmux new-session -A -s main'
alias rsyncp 'rsync -avh --progress --partial'
alias party 'echo "🎉🎉🎉 party time 🎉🎉🎉"'
alias cjk 'echo "日本語のテスト 中文测试 한국어 테스트"'
alias envprefix 'env LC_ALL=C TZ=UTC sort'
alias andor 'true; and echo yes; or echo no'
alias cmdsubst='echo (date) is today'
alias semicolons 'cd /tmp; ls; cd -'
alias longcmd 'find . -type f \( -name "*.go" -o -name "*.rs" -o -name "*.py" -o -name "*.ts" -o -name "*.js" \) -not -path "*/node_modules/*" -not -path "*/target/*" -not -path "*/.git/*" -not -path "*/vendor/*" -print0 | xargs -0 wc -l | sort -rn | head -n 25'

# abbreviations are not aliases (see the fish abbr issue)
abbr -a gco git checkout
abbr -a gcm git commit -m
abbr -a dcu docker compose up -d
abbr -a --position anywhere -- -h --help

if status is-interactive
    starship init fish | source
    zoxide init fish | source
end
