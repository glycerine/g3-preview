package symbol

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSymbolReadsRetainedSourceAssertionSwitches(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	if err != nil {
		t.Fatal(err)
	}
	d, err := diagnostic.NewDebugContext(region, budget)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Budget: budget, Debug: d})
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.Intern("a")
	if err != nil {
		t.Fatal(err)
	}
	ab, err := m.Intern("ab")
	if err != nil {
		t.Fatal(err)
	}
	ac, err := m.Intern("ac")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		left, right Symbol
		want        bool
	}{
		{a, Numerical(1), false},
		{ab, ac, true},
		{ac, ab, false},
		{a, ab, true},
		{ab, a, false},
	} {
		got, e := m.Less(tc.left, tc.right)
		if e != nil || got != tc.want {
			t.Fatal("source symbol order", tc, got, e)
		}
	}
	if text, e := m.String(a); e != nil || text != "a" {
		t.Fatal("source str", text, e)
	}
	if found, e := m.Contains(a, 'a'); e != nil || !found {
		t.Fatal("source contains", found, e)
	}
	if size, e := m.Size(a); e != nil || size != 1 {
		t.Fatal("source size", size, e)
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		left, right Symbol
		want        bool
	}{{a, Numerical(1), false}, {a, ab, true}} {
		got, e := m.Less(tc.left, tc.right)
		if sourceDebug {
			if got || !errors.Is(e, arena.ErrClosed) {
				t.Fatal("source assertion switch must stop comparison", tc, got, e)
			}
		} else if e != nil || got != tc.want {
			t.Fatal("erased source assertion switch", tc, got, e)
		}
	}
	text, textErr := m.String(a)
	found, containsErr := m.Contains(a, 'a')
	size, sizeErr := m.Size(a)
	if sourceDebug {
		if text != "" || found || size != 0 || !errors.Is(textErr, arena.ErrClosed) || !errors.Is(containsErr, arena.ErrClosed) || !errors.Is(sizeErr, arena.ErrClosed) {
			t.Fatal("source string/query assertion switches must stop", text, found, size, textErr, containsErr, sizeErr)
		}
	} else if text != "a" || !found || size != 1 || textErr != nil || containsErr != nil || sizeErr != nil {
		t.Fatal("erased string/query assertion switches", text, found, size, textErr, containsErr, sizeErr)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	if err = region.Close(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("retained symbol owner charge", budget.Used())
	}
}
