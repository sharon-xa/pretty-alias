# config.fish with no aliases at all
set -gx EDITOR vim
function fish_greeting
end
abbr -a gco git checkout
# alias commented 'ls'    <- commented out, must not count
