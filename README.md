# pretty-alias

A CLI tool to display shell aliases with syntax highlighting, supporting `bash`, `zsh`, and `fish` shells.

## Features

- Detects user's shell and reads the appropriate config file.
- Displays shell aliases in a scrollable table with syntax highlighting.
- Supports `bash`, `zsh`, and `fish` shells.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/sharon-xa/pretty-alias/main/install.sh | sudo sh
```

## Usage

Run the executable:
```sh
pretty-alias
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
