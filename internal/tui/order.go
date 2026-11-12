package tui

import (
	"cmp"
	"slices"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
)

// SetOrder sorts the table (see alias.Order). The filter keeps applying, and
// the cursor stays on the alias it was on.
func (m *Model) SetOrder(o alias.Order) {
	m.order = o
	selected := -1
	if m.cursor < len(m.rows) {
		selected = m.rows[m.cursor].idx
	}

	slices.SortFunc(m.all, func(a, b row) int {
		return cmp.Or(cmp.Compare(o.Key(a.name, a.text), o.Key(b.name, b.text)), a.idx-b.idx)
	})
	m.rows = filterRows(m.all, m.query)
	if !m.ready {
		return
	}
	m.setContent()
	m.moveCursor(slices.IndexFunc(m.rows, func(r row) bool { return r.idx == selected }))
}
