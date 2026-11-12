package tui

import "charm.land/bubbles/v2/key"

// keyMap is the single source of truth for key bindings. Update and the footer
// and `?` help all read from it.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Quit     key.Binding

	// Copy puts the selected row's command on the clipboard, CopyName its name.
	Copy     key.Binding
	CopyName key.Binding

	// Help toggles the full help; CloseHelp is esc, which also closes it.
	Help      key.Binding
	CloseHelp key.Binding

	// Filter opens the filter input; ClearFilter drops an active filter and
	// is only enabled while one is applied.
	Filter      key.Binding
	ClearFilter key.Binding

	// While the filter input has focus every printable key is text, so only
	// these are commands.
	FilterAccept    key.Binding
	FilterCancel    key.Binding
	FilterForceQuit key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next")),
		PageUp: key.NewBinding(key.WithKeys("pgup", "b"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f", "space"),
			key.WithHelp("pgdn", "page down"),
		),
		HalfUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "half page up")),
		HalfDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "half page down")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		Copy:     key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y", "copy")),
		CopyName: key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy name")),

		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		CloseHelp: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close help"),
			key.WithDisabled(),
		),

		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
			key.WithDisabled(),
		),

		FilterAccept:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "keep")),
		FilterCancel:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
		FilterForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	}
}

// ShortHelp implements help.KeyMap (the one-line footer). It is kept short
// enough to fit in 60 columns; movement keys are only in the full help.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Filter, k.ClearFilter, k.Copy, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap (the `?` view). It fits in 80 columns.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.HalfUp, k.HalfDown, k.Top, k.Bottom},
		{k.Copy, k.CopyName, k.Filter, k.ClearFilter},
		{k.Help, k.Quit},
	}
}

// filterHelp is the footer help while the filter input has focus.
func (k keyMap) filterHelp() []key.Binding {
	return []key.Binding{k.FilterAccept, k.FilterCancel}
}
