package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestDatatypeUtilCachesPinnedPreferencesPerInstance(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "utility-enum"), Constructors: []DatatypeConstructorDefinition{
		{Name: astTestSymbol(t, m, "utility-a"), Recognizer: astTestSymbol(t, m, "utility-is-a")},
		{Name: astTestSymbol(t, m, "utility-b"), Recognizer: astTestSymbol(t, m, "utility-is-b")},
		{Name: astTestSymbol(t, m, "utility-c"), Recognizer: astTestSymbol(t, m, "utility-is-c")},
	}}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	astTestPin(t, m, sorts[0])
	astTestPin(t, m, constructors[1])
	_, _, err = m.DatatypeNonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	beforeSort, err := m.GetRefCount(sorts[0])
	astTestMust(t, err)
	beforeConstructor, err := m.GetRefCount(constructors[1])
	astTestMust(t, err)
	a, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	copied := *a
	if copied.Close() == nil {
		t.Fatal("copied utility accepted ownership of original pins")
	}
	t.Cleanup(func() { astTestMust(t, a.Close()) })
	for i := 0; i < 4; i++ {
		c, found, err := a.NonRecursiveConstructor(sorts[0])
		astTestMust(t, err)
		if !found || c != constructors[1] || a.start != 1 || len(a.refs) != 2 {
			t.Fatal("preferred constructor did not remain cached", c, a.start)
		}
		_, _, err = m.DatatypeNonRecursiveConstructor(sorts[0])
		astTestMust(t, err)
	}
	b, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	c, found, err := b.NonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	if !found || c != constructors[1] || b.start != 1 {
		t.Fatal("utilities shared a selection counter")
	}
	for _, entry := range []struct {
		h      ASTHandle
		before uint32
	}{{sorts[0], beforeSort}, {constructors[1], beforeConstructor}} {
		n, err := m.GetRefCount(entry.h)
		astTestMust(t, err)
		if n != entry.before+2 {
			t.Fatal("utility cache did not own both pins", n, entry.before)
		}
	}
	astTestMust(t, a.Close())
	if _, _, err := a.NonRecursiveConstructor(sorts[0]); err == nil {
		t.Fatal("closed utility accepted a query")
	}
	astTestMust(t, b.Close())
	for _, entry := range []struct {
		h      ASTHandle
		before uint32
	}{{sorts[0], beforeSort}, {constructors[1], beforeConstructor}} {
		n, err := m.GetRefCount(entry.h)
		astTestMust(t, err)
		if n != entry.before {
			t.Fatal("utility close retained AST pins", n, entry.before)
		}
	}
}

func TestDatatypeUtilStoppedPinPrefixesRetainSourceOwners(t *testing.T) {
	for _, prefix := range []uint32{0, 1} {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			m := astTestManager(t)
			_, err := m.RegisterDatatypePlugin()
			astTestMust(t, err)
			sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "stopped-utility-sort"), Constructors: []DatatypeConstructorDefinition{
				{Name: astTestSymbol(t, m, "stopped-utility-a"), Recognizer: astTestSymbol(t, m, "stopped-is-a")},
				{Name: astTestSymbol(t, m, "stopped-utility-b"), Recognizer: astTestSymbol(t, m, "stopped-is-b")},
			}}})
			astTestMust(t, err)
			_, err = m.DatatypeConstructors(sorts[0])
			astTestMust(t, err)
			// Exhaust free reference records and place the page boundary at
			// the requested sort/declaration pin, without changing algorithms.
			var pins []ASTRef
			t.Cleanup(func() {
				for i := len(pins) - 1; i >= 0; i-- {
					astTestMust(t, m.DelRef(&pins[i]))
				}
			})
			for m.freeReference != 0 || (m.nextReference-1+prefix)%astPageSize != 0 {
				r, err := m.NewRef(sorts[0])
				astTestMust(t, err)
				pins = append(pins, r)
			}
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			before, err := m.GetRefCount(sorts[0])
			astTestMust(t, err)
			m.BudgetField.FailAfter(0)
			_, _, err = u.NonRecursiveConstructor(sorts[0])
			m.BudgetField.FailAfter(-1)
			if !errors.Is(err, arena.ErrInjectedFailure) || len(u.refs) != int(prefix) || len(u.constructors) != 0 {
				t.Fatal("failed preference lost source pin prefix or published cache", err)
			}
			after, err := m.GetRefCount(sorts[0])
			astTestMust(t, err)
			if before+prefix != after {
				t.Fatal("failed preference did not retain its acquired sort pin", before, after)
			}
			_, found, err := u.NonRecursiveConstructor(sorts[0])
			astTestMust(t, err)
			if !found || len(u.refs) != 2+int(prefix) {
				t.Fatal("stopped preference could not retry")
			}
		})
	}
}
