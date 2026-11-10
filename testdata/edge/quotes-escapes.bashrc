# Quoting forms (issue #11). The line ABOVE each alias says what the command column should show.
# (No trailing comments on alias lines here, so this file isolates quoting from issue #12.)

# expect: echo double
alias dq="echo double"
# expect: echo single
alias sq='echo single'
# expect: ls
alias bare=ls
# expect: echo "hi there"
alias nested_dq_in_sq='echo "hi there"'
# expect: echo 'hi there'
alias nested_sq_in_dq="echo 'hi there'"
# expect: "hi"   (strings.Trim strips BOTH quote pairs today)
alias quoted_quote='"hi"'
# expect: 'hi'
alias double_quoted_quote="'hi'"
# expect: echo "escaped"
alias escaped_dq="echo \"escaped\""
# expect: echo $HOME
alias escaped_dollar="echo \$HOME"
# expect: echo it's
alias sq_escape='echo it'\''s'
# expect: echo tab<TAB>here
alias ansi_c=$'echo tab\there'
# expect: echo $HOME
alias dollar_in_sq='echo $HOME'
# expect: echo a\\b   (single quotes: backslashes are literal)
alias backslash='echo a\\b'
# expect: unterminated quote: must not crash
alias mismatch="echo mismatch'
# expect: (empty command)
alias empty=''
# expect: (empty command)
alias empty2=""
# expect: lone quote char: must not crash
alias just_quote="
# expect: trailing space preserved or trimmed consistently
alias trailing_space='ls '
# expect: leading space preserved or trimmed consistently
alias leading_space=' ls'
# expect: invalid in bash: skipped or shown, but must not crash
alias spaces_around_eq = 'invalid in bash'
