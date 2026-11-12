package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/sharon-xa/pretty-alias/v2/internal/alias"
)

func orderFixture() []alias.Alias {
	return []alias.Alias{
		{Name: "gs", Command: "git status"},
		{Name: "LL", Command: "ls -la"},
		{Name: "dps", Command: "docker ps"},
		{Name: "ga", Command: "git add"},
	}
}

func rowNames(rows []row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.name
	}
	return out
}

func TestSortKeyCycles(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30))
	steps := []struct {
		order  alias.Order
		names  []string
		status string
	}{
		{alias.ByName, []string{"dps", "ga", "gs", "LL"}, "· by name"},
		{alias.ByCommand, []string{"dps", "ga", "gs", "LL"}, "· by command"},
		{alias.ByFile, []string{"gs", "LL", "dps", "ga"}, ""},
	}
	for _, st := range steps {
		send(m, press("s"))
		if m.order != st.order || !slices.Equal(rowNames(m.rows), st.names) {
			t.Errorf("after s: order %s rows %v, want %s %v",
				m.order, rowNames(m.rows), st.order, st.names)
		}
		line := lastLine(m)
		if st.status != "" && !strings.Contains(line, st.status) {
			t.Errorf("%s: footer lacks %q: %q", st.order, st.status, line)
		}
		if st.status == "" && strings.Contains(line, "· by ") {
			t.Errorf("file order shown in footer: %q", line)
		}
	}
}

// The table on screen follows the order, not just m.rows.
func TestSortReordersTheTable(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30), press("s"))
	v := plain(m)
	if strings.Index(v, "dps") > strings.Index(v, "gs") {
		t.Errorf("table not re-rendered in name order:\n%s", v)
	}
}

func TestSortKeepsCursorOnSameAlias(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30), press("j")) // LL
	for range 3 {
		send(m, press("s"))
		if got := m.rows[m.cursor].name; got != "LL" {
			t.Fatalf("order %s: cursor on %q, want LL", m.order, got)
		}
		assertCursorVisible(t, m)
	}
}

func TestSortWithFilter(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30), press("/"))
	typeText(m, "git")
	send(m, press("enter"), press("s")) // by name
	if got := rowNames(m.rows); !slices.Equal(got, []string{"ga", "gs"}) {
		t.Errorf("filtered + by name: %v", got)
	}
	send(m, press("esc")) // clear the filter: the order stays
	if got := rowNames(m.rows); !slices.Equal(got, []string{"dps", "ga", "gs", "LL"}) {
		t.Errorf("after clearing the filter: %v", got)
	}
}

// s is a letter while typing a filter.
func TestSortKeyIsTextWhileFiltering(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30), press("/"), press("s"))
	if m.order != alias.ByFile || m.filter.Value() != "s" {
		t.Errorf("order %s, query %q", m.order, m.filter.Value())
	}
}

// --sort sets the starting order before the first frame; s still cycles from it
// and gets back to the real file order.
func TestSetOrderBeforeStart(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	m.SetOrder(alias.ByCommand)
	send(m, size(100, 30))
	if got := rowNames(m.rows); !slices.Equal(got, []string{"dps", "ga", "gs", "LL"}) {
		t.Errorf("by command: %v", got)
	}
	send(m, press("s"))
	if got := rowNames(m.rows); !slices.Equal(got, []string{"gs", "LL", "dps", "ga"}) {
		t.Errorf("back to file order: %v", got)
	}
}

func TestSortInFullHelp(t *testing.T) {
	m := New(orderFixture(), nil, "x")
	send(m, size(100, 30), press("?"))
	if !strings.Contains(plain(m), "s sort") {
		t.Errorf("? help lacks the sort key:\n%s", plain(m))
	}
}

func TestSortWithNoRows(t *testing.T) {
	m := New(nil, nil, "x")
	send(m, size(80, 24), press("s"), press("s"), press("j"))
	_ = m.content()
}
