if status is-interactive
    alias ls 'eza --icons'
    alias ll 'eza -la --icons'
    if command -q bat
        alias cat 'bat --style=plain'
    end
end
alias top_level 'echo not-indented'
