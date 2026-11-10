# Option flags (issue #10). Global/suffix aliases are zsh-specific.
alias -g L='| less'
alias -g G='| grep'
alias -g H='| head'
alias -g NE='2>/dev/null'
alias -s txt=nvim
alias -s md=glow
alias -s pdf=zathura
alias -s {jpg,png}=imv
alias -- -='cd -'
alias -- --help='echo not-a-flag'
alias ll='ls -la'
