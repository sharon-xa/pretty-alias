package alias

import (
	"slices"
	"testing"
)

func TestParseOrder(t *testing.T) {
	for _, o := range Orders {
		if got, err := ParseOrder(string(o)); err != nil || got != o {
			t.Errorf("ParseOrder(%q) = %q, %v", o, got, err)
		}
	}
	for _, s := range []string{"", "Name", "alpha", "files"} {
		if _, err := ParseOrder(s); err == nil {
			t.Errorf("ParseOrder(%q) accepted", s)
		}
	}
}

func TestOrderNextCycles(t *testing.T) {
	got := []Order{ByFile}
	for range 3 {
		got = append(got, got[len(got)-1].Next())
	}
	if want := []Order{ByFile, ByName, ByCommand, ByFile}; !slices.Equal(got, want) {
		t.Errorf("cycle = %v, want %v", got, want)
	}
}

func names(as []Alias) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Name
	}
	return out
}

func TestSort(t *testing.T) {
	file := []Alias{
		{Name: "gs", Command: "git status"},
		{Name: "LL", Command: "ls -la"},
		{Name: "dps", Command: "docker ps"},
		{Name: "ga", Command: "git add"},
		{Name: "la", Command: "ls -A"},
		{Name: "g2", Command: "git status"}, // same command as gs: keeps file order
	}
	tests := []struct {
		o    Order
		want []string
	}{
		{ByFile, []string{"gs", "LL", "dps", "ga", "la", "g2"}},
		{ByName, []string{"dps", "g2", "ga", "gs", "la", "LL"}},    // case ignored
		{ByCommand, []string{"dps", "ga", "gs", "g2", "la", "LL"}}, // "ls -a" < "ls -la"
	}
	for _, tt := range tests {
		as := slices.Clone(file)
		Sort(as, tt.o)
		if got := names(as); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %v, want %v", tt.o, got, tt.want)
		}
	}
}
