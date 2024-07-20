# pretty-alias

A CLI tool to display shell aliases with syntax highlighting, supporting `bash`, `zsh`, and `fish` shells.

## Features

- Detects user's shell and reads the appropriate config file.
- Displays shell aliases in a scrollable table with syntax highlighting.
- Supports `bash`, `zsh`, and `fish` shells.

## Installation

```bash
sh -c "$(curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh)"
```

## Usage

Run the executable:
```sh
pretty-alias
```

It's better to add this alias and use it instead of the default alias command
1. Bash
```bash
echo "alias alias='pretty-alias'" >> ~/.bashrc
```
2. zsh
```bash
echo "alias alias='pretty-alias'" >> ~/.zshrc
```
2. fish
```bash
echo "alias alias='pretty-alias'" >> ~/.config/fish/config.fish
```


Use the following keys for navigation:
- `q` or `ctrl+c`: Quit
- `up`: Scroll up
- `down`: Scroll down

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Chroma](https://github.com/alecthomas/chroma)

## License

This project is licensed under the MIT License.
