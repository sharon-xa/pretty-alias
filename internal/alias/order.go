package alias

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Order is how aliases are listed.
type Order string

// The orders, in the order the TUI's sort key cycles through them.
const (
	ByFile    Order = "file"    // as defined in the config file
	ByName    Order = "name"    // by alias name, ignoring case
	ByCommand Order = "command" // by command, ignoring case (groups "git ..." together)
)

// Orders lists every Order.
var Orders = []Order{ByFile, ByName, ByCommand}

// ParseOrder turns "file", "name" or "command" into an Order.
func ParseOrder(s string) (Order, error) {
	if o := Order(s); slices.Contains(Orders, o) {
		return o, nil
	}
	return "", fmt.Errorf("unknown sort order %q (want file, name or command)", s)
}

// Next returns the order after o, wrapping around.
func (o Order) Next() Order {
	i := slices.Index(Orders, o)
	return Orders[(i+1)%len(Orders)]
}

// Key returns what o sorts an alias by: its name or command in lower case, or
// "" for ByFile. Callers break ties by file position, so equal keys keep the
// file's order.
func (o Order) Key(name, command string) string {
	switch o {
	case ByName:
		return strings.ToLower(name)
	case ByCommand:
		return strings.ToLower(command)
	}
	return ""
}

// Sort orders as by o. It is stable, so ByFile leaves a slice that is in file
// order unchanged, and aliases with equal keys keep their relative order.
func Sort(as []Alias, o Order) {
	slices.SortStableFunc(as, func(a, b Alias) int {
		return cmp.Compare(o.Key(a.Name, a.Command), o.Key(b.Name, b.Command))
	})
}
