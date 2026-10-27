package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestDatatypeEnumCacheOwnsConstructorsAndBooleanResults(t *testing.T) {
	m := astTestManager(t)
	enum, nat := datatypeSelectionEnum(t, m), datatypeRecursionNat(t, m)
	handles := []ASTHandle{enum, nat}
	// Constructor declarations themselves own their range sorts. Keep them
	// alive independently so only utility pins vary below.
	for _, h := range handles {
		constructors, err := m.DatatypeConstructors(h)
		astTestMust(t, err)
		for _, constructor := range constructors {
			astTestPin(t, m, constructor)
		}
	}
	before := make([]uint32, 2)
	for i, h := range handles {
		var err error
		before[i], err = m.GetRefCount(h)
		astTestMust(t, err)
	}
	a, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, a.Close()) })
	b, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	for _, u := range []*DatatypeUtil{a, b} {
		for repeat := 0; repeat < 3; repeat++ {
			for i, h := range handles {
				value, err := u.IsEnumSort(h)
				astTestMust(t, err)
				if value != (i == 0) {
					t.Fatal("wrong enumeration classification", i, value)
				}
			}
		}
		if len(u.enum) != 2 || len(u.lists) != 2 || len(u.refs) != 9 || u.start != 0 || len(u.constructors) != 0 {
			t.Fatal("enum predicate lost result/list owners or changed preference", len(u.refs))
		}
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i]+4 {
			t.Fatal("missing two sort pins per enum utility", i, count, before[i])
		}
	}
	astTestMust(t, a.Reset())
	if len(a.enum) != 0 || len(a.lists) != 0 || len(a.refs) != 0 || len(b.enum) != 2 {
		t.Fatal("Reset did not isolate enum utility owners")
	}
	astTestMust(t, b.Close())
	if b.enum != nil {
		t.Fatal("Close retained enum keys")
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i] {
			t.Fatal("enum cleanup retained sort pin", i, count, before[i])
		}
	}
}

func TestDatatypeEnumCachePublishesBeforeResultPinRefusal(t *testing.T) {
	for _, isEnum := range []bool{false, true} {
		t.Run(map[bool]string{false: "non-enum", true: "enum"}[isEnum], func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			if !isEnum {
				sort = datatypeRecursionNat(t, m)
			}
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			var pins []ASTRef
			t.Cleanup(func() {
				astTestMust(t, u.Close())
				for i := range pins {
					astTestMust(t, m.DelRef(&pins[i]))
				}
			})
			_, err = u.Constructors(sort)
			astTestMust(t, err)
			beforeRefs := len(u.refs)
			for m.freeReference != 0 || (m.nextReference-1)%astPageSize != 0 {
				pin, err := m.NewRef(sort)
				astTestMust(t, err)
				pins = append(pins, pin)
			}
			m.BudgetField.FailAfter(0)
			_, err = u.IsEnumSort(sort)
			m.BudgetField.FailAfter(-1)
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal("missing enum result pin refusal", err)
			}
			result, found := u.enum[sort]
			if !found || result != isEnum || len(u.refs) != beforeRefs {
				t.Fatal("result pin refusal lost preceding result or constructor owners", result, found, len(u.refs))
			}
			m.BudgetField.FailAfter(0)
			result, err = u.IsEnumSort(sort)
			m.BudgetField.FailAfter(-1)
			astTestMust(t, err)
			if result != isEnum || len(u.refs) != beforeRefs {
				t.Fatal("warm enum lookup retried failed pin")
			}
		})
	}
}

func TestDatatypeEnumPredicateSourceGatesAndWarmCache(t *testing.T) {
	for _, mode := range []string{"non-datatype", "cold", "warm-true", "warm-false"} {
		t.Run(mode, func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			if mode == "warm-false" {
				sort = datatypeRecursionNat(t, m)
			}
			if mode == "non-datatype" {
				var err error
				sort, err = m.MkBoolSort()
				astTestMust(t, err)
			}
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			if mode == "warm-true" || mode == "warm-false" {
				_, err = u.IsEnumSort(sort)
				astTestMust(t, err)
			}
			_, _, d, _, _ := astDebugActionFixture(t, "T")
			previous, err := diagnostic.BindProcessDebugContext(d)
			astTestMust(t, err)
			t.Cleanup(func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) })
			astTestMust(t, d.Close())
			value, err := u.IsEnumSort(sort)
			if mode == "cold" && diagnostic.DebugBuild {
				if !errors.Is(err, arena.ErrClosed) {
					t.Fatal("cold enum query skipped source constructor assertion", err)
				}
			} else {
				astTestMust(t, err)
				if value != (mode == "cold" || mode == "warm-true") {
					t.Fatal("enum gate/warm cache mismatch", mode, value)
				}
			}
			if mode == "non-datatype" && (len(u.enum) != 0 || len(u.lists) != 0 || len(u.refs) != 0) {
				t.Fatal("non-datatype predicate published owners")
			}
		})
	}
}
