# TODO

The full backlog lives in GitHub issues; see the roadmap issue (#45) and the milestones. This file is only a short pointer list.

- [x] Resizing the terminal window is causing a crash.
- [x] Show something when there are no aliases (message on stderr, exit 0).
- [x] Test the CLI with fish, zsh and bash, with 0 / some / lots of aliases (`testdata/`, `just smoke`, `go test ./...`).
- [ ] Create a way to categorize aliases (#38):
    - You can create a syntax for pretty-alias to look for in aliases file.
    - Separate them visually when there are categories.
- [ ] Follow `source`d alias files such as `~/.bash_aliases` (#2, #31).
