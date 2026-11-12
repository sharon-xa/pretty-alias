package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
)

// keyMap is the single source of truth for key bindings. The footer help and
// the viewport's scrolling both read from it, so adding a binding here is all
// it takes to make it work and show up in `?`-style help later.
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
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
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

// ShortHelp implements help.KeyMap (the one-line footer).
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.PageDown, k.Top, k.Filter, k.ClearFilter, k.Quit}
}

// FullHelp implements help.KeyMap (reserved for a future `?` overlay).
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.HalfUp, k.HalfDown, k.Top, k.Bottom},
		{k.Filter, k.ClearFilter, k.Quit},
	}
}

// filterHelp is the footer help while the filter input has focus.
func (k keyMap) filterHelp() []key.Binding {
	return []key.Binding{k.FilterAccept, k.FilterCancel}
}

// viewportKeys maps our bindings onto the viewport component.
func (k keyMap) viewportKeys() viewport.KeyMap {
	return viewport.KeyMap{
		Up:           k.Up,
		Down:         k.Down,
		PageUp:       k.PageUp,
		PageDown:     k.PageDown,
		HalfPageUp:   k.HalfUp,
		HalfPageDown: k.HalfDown,
		// Left/Right are for horizontal scrolling, which we don't use.
		Left:  key.NewBinding(key.WithDisabled()),
		Right: key.NewBinding(key.WithDisabled()),
	}
}
