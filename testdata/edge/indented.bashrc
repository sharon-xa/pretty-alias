# Aliases inside blocks are indented (issue #9, probably #1). Expected rows: 7
if [ -x /usr/bin/dircolors ]; then
    alias ls='ls --color=auto'
    alias grep='grep --color=auto'
fi

case "$OSTYPE" in
  darwin*)
	alias open='open'          # tab-indented
	alias update='brew update && brew upgrade'
	;;
  linux*)
    alias open='xdg-open'
    alias update='sudo apt update && sudo apt upgrade'
	;;
esac

alias top_level='echo i-am-not-indented'
