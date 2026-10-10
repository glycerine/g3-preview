package sexpr

import (
	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"testing"
)

func symbolManagerTest(t *testing.T, b *arena.Budget, direct bool) *usymbol.Manager {
	t.Helper()
	m, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2, DirectOnly: direct})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Fatal(e)
		}
	})
	return m
}
func internSymbol(t *testing.T, m *usymbol.Manager, text string) usymbol.Symbol {
	t.Helper()
	s, e := m.Intern(text)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func symbolText(t *testing.T, m *usymbol.Manager, s usymbol.Symbol) string {
	t.Helper()
	v, e := m.String(s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
