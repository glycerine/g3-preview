package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func datatypeRecursionNat(t *testing.T, m *ASTManager) ASTHandle {
	t.Helper()
	sorts, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "RecursionNat"), Constructors: []DatatypeConstructorDefinition{
		{Name: astTestSymbol(t, m, "recursion-zero"), Recognizer: astTestSymbol(t, m, "is-recursion-zero")},
		{Name: astTestSymbol(t, m, "recursion-succ"), Recognizer: astTestSymbol(t, m, "is-recursion-succ"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "recursion-pred"), Recursive: true}}},
	}}})
	astTestMust(t, err)
	astTestPin(t, m, sorts[0])
	return sorts[0]
}

func TestDatatypeRecursionCacheIndependentPinsAndReset(t *testing.T) {
	m := astTestManager(t)
	enum, nat := datatypeSelectionEnum(t, m), datatypeRecursionNat(t, m)
	handles := []ASTHandle{enum, nat}
	before := make([]uint32, len(handles))
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
	copy := *a
	if _, err := copy.IsRecursive(enum); !errors.Is(err, ErrASTHandle) {
		t.Fatal("copied recursion owner accepted query", err)
	}
	for _, u := range []*DatatypeUtil{a, b} {
		for repeat := 0; repeat < 3; repeat++ {
			for i, h := range handles {
				got, err := u.IsRecursive(h)
				astTestMust(t, err)
				if got != (i == 1) {
					t.Fatal("wrong recursive result", i, got)
				}
			}
		}
		if len(u.recursive) != 2 || len(u.refs) != 2 || len(u.lists) != 0 || len(u.constructors) != 0 || u.start != 0 {
			t.Fatal("recursion queries changed unrelated caches/counter", u)
		}
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i]+2 {
			t.Fatal("missing independent recursive-result sort pins", i, count, before[i])
		}
	}
	astTestMust(t, a.Reset())
	if len(a.recursive) != 0 || len(a.refs) != 0 || a.start != 1 || len(b.recursive) != 2 {
		t.Fatal("Reset did not isolate recursion caches")
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i]+1 {
			t.Fatal("Reset retired another utility's pin", i, count)
		}
	}
	astTestMust(t, b.Close())
	if b.recursive != nil {
		t.Fatal("Close retained borrowed recursion keys")
	}
	if _, err := b.IsRecursive(enum); !errors.Is(err, ErrASTHandle) {
		t.Fatal("closed recursion cache revived", err)
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i] {
			t.Fatal("Close retained recursion pin", i, count)
		}
	}
}

func TestDatatypeRecursionCachePublishesBeforePinRefusal(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		t.Run(map[bool]string{false: "finite", true: "recursive"}[recursive], func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			if recursive {
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
			// Use only actual NewRef operations to reach the next reference-page allocation.
			for m.freeReference != 0 || (m.nextReference-1)%astPageSize != 0 {
				pin, err := m.NewRef(sort)
				astTestMust(t, err)
				pins = append(pins, pin)
			}
			before, err := m.GetRefCount(sort)
			astTestMust(t, err)
			m.BudgetField.FailAfter(0)
			_, err = u.IsRecursive(sort)
			m.BudgetField.FailAfter(-1)
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal("missing physical sort-pin refusal", err)
			}
			result, found := u.recursive[sort]
			if !found || result != recursive || len(u.refs) != 0 {
				t.Fatal("pin refusal lost preceding cache publication", result, found, len(u.refs))
			}
			count, err := m.GetRefCount(sort)
			astTestMust(t, err)
			if count != before {
				t.Fatal("refused reference changed its borrowed sort count", count, before)
			}
			m.BudgetField.FailAfter(0)
			result, err = u.IsRecursive(sort)
			m.BudgetField.FailAfter(-1)
			astTestMust(t, err)
			if result != recursive || len(u.refs) != 0 {
				t.Fatal("warm cache retried the stopped source pin")
			}
			astTestMust(t, u.Reset())
			result, err = u.IsRecursive(sort)
			astTestMust(t, err)
			if result != recursive || len(u.refs) != 1 {
				t.Fatal("Reset did not permit a fresh owned cache entry")
			}
		})
	}
}

func TestDatatypeRecursionCacheAssertionOwnerBeforeWarmLookup(t *testing.T) {
	for _, warm := range []bool{false, true} {
		t.Run(map[bool]string{false: "cold", true: "warm"}[warm], func(t *testing.T) {
			m := astTestManager(t)
			sort := datatypeSelectionEnum(t, m)
			u, err := NewDatatypeUtil(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, u.Close()) })
			if warm {
				_, err = u.IsRecursive(sort)
				astTestMust(t, err)
			}
			_, _, d, _, _ := astDebugActionFixture(t, "T")
			previous, err := diagnostic.BindProcessDebugContext(d)
			astTestMust(t, err)
			t.Cleanup(func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) })
			astTestMust(t, d.Close())
			_, err = u.IsRecursive(sort)
			if diagnostic.DebugBuild {
				if !errors.Is(err, arena.ErrClosed) {
					t.Fatal("recursion assertion ignored actual closed owner", err)
				}
			} else {
				astTestMust(t, err)
			}
		})
	}
}

func TestDatatypeRecursionSourceTraversalSharedDAGAndNestedSorts(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.RegisterDeclarationPlugins())
	names := []string{"RecRoot", "RecLeft", "RecRight", "RecLeaf"}
	edges := [][]uint32{{1, 2}, {3, 3}, {3}, nil}
	definitions := make([]DatatypeDefinition, len(names))
	for i, name := range names {
		fields := make([]DatatypeAccessorDefinition, len(edges[i]))
		for j, target := range edges[i] {
			fields[j] = DatatypeAccessorDefinition{Name: astTestSymbol(t, m, name+string(rune('a'+j))), Recursive: true, RecursiveIndex: target}
		}
		definitions[i] = DatatypeDefinition{Name: astTestSymbol(t, m, name), Constructors: []DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "mk-"+name), Recognizer: astTestSymbol(t, m, "is-"+name), Accessors: fields}}}
	}
	sorts, err := m.MkDatatypes(definitions)
	astTestMust(t, err)
	for _, sort := range sorts {
		astTestPin(t, m, sort)
	}
	nat := datatypeRecursionNat(t, m)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	array, err := m.MkArraySort(SliceArguments([]ASTHandle{boolean}), nat)
	astTestMust(t, err)
	wrapper, err := m.MkDatatypes([]DatatypeDefinition{{Name: astTestSymbol(t, m, "RecArrayWrapper"), Constructors: []DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "rec-array"), Recognizer: astTestSymbol(t, m, "is-rec-array"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "rec-array-field"), Range: array}}}}}})
	astTestMust(t, err)
	astTestPin(t, m, wrapper[0])
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	for i, sort := range append(sorts, wrapper[0]) {
		want := i == len(sorts)
		got, err := u.IsRecursive(sort)
		astTestMust(t, err)
		direct, err := m.DatatypeIsRecursive(sort)
		astTestMust(t, err)
		if got != want || direct != want {
			t.Fatal("source stack/shared descendant or nested sort recursion mismatch", i, got, direct, want)
		}
	}
}

func TestDatatypeRecursionUsesDefinitionRangesBeforeParameterSubstitution(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	nat := datatypeRecursionNat(t, m)
	formal, err := m.MkSort(astTestSymbol(t, m, "RecursionFormal"), ASTSortInfo{})
	astTestMust(t, err)
	astTestPin(t, m, formal)
	name := astTestSymbol(t, m, "RecursionBox")
	_, err = m.MkDatatypes([]DatatypeDefinition{{Name: name, Parameters: []ASTHandle{formal}, Constructors: []DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "recursion-box"), Recognizer: astTestSymbol(t, m, "is-recursion-box"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "recursion-unbox"), Range: formal}}}}}})
	astTestMust(t, err)
	p0, err := m.NewParameterSymbol(name)
	astTestMust(t, err)
	p1, err := m.NewParameterAST(nat)
	astTestMust(t, err)
	box, err := m.MkFamilySort(family, ASTDatatypeSort, []ASTParameter{p0, p1})
	astTestMust(t, err)
	astTestPin(t, m, box)
	astTestMust(t, m.DelParameter(&p1))
	astTestMust(t, m.DelParameter(&p0))
	cs, err := m.DatatypeConstructors(box)
	astTestMust(t, err)
	domain, err := m.GetDomain(cs[0], 0)
	astTestMust(t, err)
	if domain != nat {
		t.Fatal("fixture did not instantiate its recursive constructor domain")
	}
	u, err := NewDatatypeUtil(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, u.Close()) })
	recursive, err := u.IsRecursive(box)
	astTestMust(t, err)
	direct, err := m.DatatypeIsRecursive(box)
	astTestMust(t, err)
	// datatype_decl_plugin.h:278 and cpp:543 consult the generic named def.
	if recursive || direct {
		t.Fatal("recursion analysis substituted actual domains instead of reading definition ranges")
	}
}
