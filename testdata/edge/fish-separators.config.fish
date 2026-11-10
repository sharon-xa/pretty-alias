# fish accepts both 'alias NAME DEF' and 'alias NAME=DEF'. The current parser picks the FIRST of '=' / ' '.
alias a1 'echo space-separated'
alias a2='echo equals-separated'
alias a3 "echo dq space"
alias a4="echo dq equals"
alias a5 echo unquoted words
alias a6=echo
alias a7 FOO=bar env
alias a8 'FOO=bar env'
alias a9='FOO=bar env'
alias a10 'echo a=b'
alias a11='echo a=b'
alias a12 "echo 'a = b'"
alias a13  'two spaces after name'
alias a14	'tab after name'
alias a15 ' leading space in value'
alias a16 'trailing space in value '
alias a17 ''
alias a18 ""
