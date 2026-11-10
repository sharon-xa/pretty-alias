# includes itself and the other file: a naive follower would loop forever
alias from_self_include='echo once'
source "$HOME/.config/aliases/self-include.sh"
source "$HOME/.config/aliases/work.sh"
