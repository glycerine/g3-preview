package ast

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func datatypeSelectionEnum(t *testing.T, m *ASTManager) ASTHandle {
	t.Helper()
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "selection-enum"), Constructors: []DatatypeConstructorDefinition{
		{Name: astTestSymbol(t, m, "selection-a"), Recognizer: astTestSymbol(t, m, "is-selection-a")},
		{Name: astTestSymbol(t, m, "selection-b"), Recognizer: astTestSymbol(t, m, "is-selection-b")},
		{Name: astTestSymbol(t, m, "selection-c"), Recognizer: astTestSymbol(t, m, "is-selection-c")},
	}}})
	astTestMust(t, err)
	astTestPin(t, m, sorts[0])
	return sorts[0]
}

func TestDatatypeSelectionInstantiatesAllConstructorsBeforeCounter(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "declaration-order", true: "cold-failure"}[fail], func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			if fail {
				// Fill node storage using real independent variables. The first
				// cold declaration then requires a new physical node page.
				boolean, err := m.MkBoolSort()
				astTestMust(t, err)
				for i := uint32(0); m.freeNode != 0 || (m.nextNode-1)%astPageSize != 0; i++ {
					_, err := m.MkVar(i, boolean)
					astTestMust(t, err)
				}
				m.Budget().FailAfter(0)
				_, _, err = u.NonRecursiveConstructor(sort)
				m.Budget().FailAfter(-1)
				if !errors.Is(err, arena.ErrInjectedFailure) || u.start != 0 || len(u.refs) != 0 || len(u.constructors) != 0 {
					t.Fatal("cold list failure advanced selection or published preference", err, u.start)
				}
			}
			chosen, found, err := u.NonRecursiveConstructor(sort)
			astTestMust(t, err)
			constructors, err := m.DatatypeConstructors(sort)
			astTestMust(t, err)
			if !found || chosen != constructors[1] || u.start != 1 {
				t.Fatal("source selection changed", chosen, u.start)
			}
			var last uint32
			for i, constructor := range constructors {
				id, err := m.GetASTID(constructor)
				astTestMust(t, err)
				if i != 0 && id <= last {
					t.Fatal("constructors instantiated in selection order instead of declaration order")
				}
				last = id
			}
		})
	}
}

func TestDatatypeSelectionUnsignedIndexWrap(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "wrap-list"), Constructors: []DatatypeConstructorDefinition{
		{Name: astTestSymbol(t, m, "wrap-rec-a"), Recognizer: astTestSymbol(t, m, "is-wrap-a"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "tail-a"), Recursive: true}}},
		{Name: astTestSymbol(t, m, "wrap-nil"), Recognizer: astTestSymbol(t, m, "is-wrap-nil")},
		{Name: astTestSymbol(t, m, "wrap-rec-b"), Recognizer: astTestSymbol(t, m, "is-wrap-b"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "tail-b"), Recursive: true}}},
	}}})
	astTestMust(t, err)
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	// The literal source unsigned loop visits indices 2,0,0 at this
	// boundary, skipping the available nullary constructor at index 1.
	// Reducing start before addition would incorrectly visit 2,0,1.
	u.start = math.MaxUint32 - 2
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: m.Budget()}})
	astTestMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, m.Budget())
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, debug.Close()); astTestMust(t, region.Close()) })
	previous, err := diagnostic.BindProcessDebugContext(debug)
	astTestMust(t, err)
	t.Cleanup(func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) })
	value, found, err := u.NonRecursiveConstructor(sorts[0])
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("missing source r assertion", err)
		}
	} else {
		astTestMust(t, err)
	}
	if found || !value.IsNull() || u.start != math.MaxUint32-1 || len(u.refs) != 0 || len(u.constructors) != 0 {
		t.Fatal("unsigned source candidate sequence changed", value, found, u.start)
	}
}

func TestDatatypeUtilSourceAssertionsConsultActualOwner(t *testing.T) {
	for _, site := range []string{"constructor", "cold-query", "warm-query"} {
		t.Run(site, func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			if site == "warm-query" {
				_, _, err = u.NonRecursiveConstructor(sort)
				astTestMust(t, err)
			}
			_, _, d, _, _ := astDebugActionFixture(t, "T")
			previous, err := diagnostic.BindProcessDebugContext(d)
			astTestMust(t, err)
			t.Cleanup(func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) })
			astTestMust(t, d.Close())
			if site == "constructor" {
				var other *DatatypeUtil
				other, err = NewDatatypeUtil(m)
				if other != nil {
					astTestMust(t, other.Close())
				}
			} else {
				_, _, err = u.NonRecursiveConstructor(sort)
			}
			if diagnostic.DebugBuild {
				if !errors.Is(err, arena.ErrClosed) {
					t.Fatal("reached source assertion ignored closed owner", err)
				}
			} else {
				astTestMust(t, err)
			}
		})
	}
}
