// Source: src/ast/array_decl_plugin.cpp:25-114; src/ast/array_decl_plugin.h:67-130.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTArraySourceConstructorAndInfiniteSortParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	family, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	domain, e := m.MkFamilySort(m.arithmeticPlugin.family, 1, nil)
	astTestMust(t, e)
	rangeSort, e := m.MkBoolSort()
	astTestMust(t, e)
	dp, e := m.NewParameterAST(domain)
	astTestMust(t, e)
	rp, e := m.NewParameterAST(rangeSort)
	astTestMust(t, e)
	array, e := m.MkFamilySort(family, 0, []ASTParameter{dp, rp})
	astTestMust(t, e)
	r, e := m.internal(array)
	astTestMust(t, e)
	info, e := m.sortInfo(m.sortAt(r.slot).info)
	astTestMust(t, e)
	if info.family != family || info.kind != 0 || info.count != 2 || !info.size.IsInfinite() {
		t.Fatal("literal array sort", *info)
	}
	for j, want := range []ASTHandle{domain, rangeSort} {
		p, e := m.infoParameter(info, uint32(j))
		astTestMust(t, e)
		record, e := m.parameter(p)
		value := m.external(record.ast)
		astTestMust(t, e)
		if value != want {
			t.Fatal("actual copied parameters", j, value, want)
		}
	}
	astTestMust(t, m.DelParameter(&rp))
	astTestMust(t, m.DelParameter(&dp))
	for i, text := range []string{"store", "select", "const", "default", "map", "union", "intersect", "difference", "complement", "subset", "array-ext", "as-array"} {
		actual, e := m.Symbols.String(m.arrayPlugin.names[i])
		astTestMust(t, e)
		if string(actual) != text {
			t.Fatal("source constructor order", i, string(actual), text)
		}
	}
}

func TestASTArrayEverySymbolConstructorAndPublicationReservation(t *testing.T) {
	_, q, _ := astTestOwners(t)
	completed, failed := false, false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 28)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterArrayPlugin()
		if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
			t.Fatal("immediate actual constructor owners", ordinal, b.Used(), m.Stats(), sy.Stats())
		}
		if call != nil {
			failed = true
			if !errors.Is(call, arena.ErrInjectedFailure) {
				t.Fatal(ordinal, call)
			}
			for p := m.arrayPluginFailed; p != nil; p = p.next {
				if p.published || p.completed > 12 || p.bodyCause == nil {
					t.Fatal("unfinished source wrapper", ordinal, p.completed, p.bodyCause)
				}
			}
		} else {
			completed = true
			if !m.arrayPlugin.published || m.arrayPlugin.completed != 12 {
				t.Fatal("all literal completed members before publication", ordinal, m.arrayPlugin)
			}
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("real owning constructor end", ordinal, b.Used())
		}
		if completed {
			t.Log("first full constructor success", ordinal)
			break
		}
	}
	if !completed || !failed {
		t.Fatal("constructor horizon", completed, failed)
	}
}

func astArrayFiniteTestSort(t *testing.T, m *ASTManager, n uint64, label string) ASTHandle {
	t.Helper()
	name, e := m.Symbols.Intern(label)
	astTestMust(t, e)
	family, e := m.families.MkID(name)
	astTestMust(t, e)
	info, e := m.NewSortInfo(family, 0, nil, FiniteSortSize(n), false)
	astTestMust(t, e)
	h, e := m.MkSort(name, info)
	astTestMust(t, e)
	astTestMust(t, m.DelSortInfo(&info))
	return h
}
func TestASTArrayFiniteCardinalityIndependentBoundedOracle(t *testing.T) {
	m := astTestManager(t)
	family, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	for _, tc := range []struct {
		domains   []uint64
		rangeSize uint64
		want      uint64
		veryBig   bool
	}{
		{[]uint64{2}, 3, 9, false}, {[]uint64{2, 3}, 2, 64, false}, {[]uint64{0}, 0, 1, false},
		{[]uint64{128}, 2, 0, true}, {[]uint64{129}, 2, 0, true}, {[]uint64{^uint64(0), ^uint64(0)}, 1, 1, false},
	} {
		var ps []ASTParameter
		for i, n := range append(append([]uint64{}, tc.domains...), tc.rangeSize) {
			h := astArrayFiniteTestSort(t, m, n, fmt.Sprintf("finite-%d-%d", i, n))
			p, e := m.NewParameterAST(h)
			astTestMust(t, e)
			ps = append(ps, p)
		}
		h, e := m.MkFamilySort(family, 0, ps)
		astTestMust(t, e)
		r, e := m.internal(h)
		astTestMust(t, e)
		info, e := m.sortInfo(m.sortAt(r.slot).info)
		astTestMust(t, e)
		if tc.veryBig {
			if !info.size.IsVeryBig() {
				t.Fatal("literal cardinality cutoff", tc, info.size)
			}
		} else {
			actual, e := info.size.Size()
			astTestMust(t, e)
			if actual != tc.want {
				t.Fatal("independent bounded count", tc, actual)
			}
		}
		for i := range ps {
			astTestMust(t, m.DelParameter(&ps[i]))
		}
	}
}

func TestASTArraySetSortActualLocalParameterCopy(t *testing.T) {
	m := astTestManager(t)
	family, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	domain := astArrayFiniteTestSort(t, m, 8, "set-eight-domain")
	p, e := m.NewParameterAST(domain)
	astTestMust(t, e)
	h, e := m.MkFamilySort(family, 1, []ASTParameter{p})
	astTestMust(t, e)
	r, e := m.internal(h)
	astTestMust(t, e)
	info, e := m.sortInfo(m.sortAt(r.slot).info)
	astTestMust(t, e)
	n, e := info.size.Size()
	astTestMust(t, e)
	if n != 256 || info.kind != 0 || info.count != 2 {
		t.Fatal("SET -> ARRAY source branch", n, *info)
	}
	want := []astNodeRef{{}, m.boolSort}
	want[0], e = m.internal(domain)
	astTestMust(t, e)
	for i := uint32(0); i < 2; i++ {
		copy, e := m.infoParameter(info, i)
		astTestMust(t, e)
		value, e := m.parameter(copy)
		astTestMust(t, e)
		if value.Kind != ASTParameterAST || value.ast != want[i] {
			t.Fatal("actual distinct stack parameter reads", i, *value, want[i])
		}
	}
	astTestMust(t, m.DelParameter(&p))
	again, e := m.infoParameter(info, 0)
	astTestMust(t, e)
	value, e := m.parameter(again)
	astTestMust(t, e)
	if value.ast != want[0] {
		t.Fatal("owned copy outlives caller parameter", *value)
	}
}

// Source: src/ast/array_decl_plugin.cpp:86-114. Enumerating finite functions
// supplies an independent bounded oracle rather than reproducing rational power.
func TestASTArrayFiniteFunctionEnumerationOracle(t *testing.T) {
	m := astTestManager(t)
	family, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	for left := uint64(0); left <= 3; left++ {
		for right := uint64(1); right <= 3; right++ {
			for rangeSize := uint64(0); rangeSize <= 3; rangeSize++ {
				positions := int(left * right)
				var enumerate func(int) uint64
				enumerate = func(i int) uint64 {
					if i == positions {
						return 1
					}
					var count uint64
					for output := uint64(0); output < rangeSize; output++ {
						count += enumerate(i + 1)
					}
					return count
				}
				want := enumerate(0)
				var ps [3]ASTParameter
				for i, n := range [3]uint64{left, right, rangeSize} {
					h := astArrayFiniteTestSort(t, m, n, fmt.Sprintf("enumerate-%d-%d-%d-%d", left, right, rangeSize, i))
					ps[i], e = m.NewParameterAST(h)
					astTestMust(t, e)
				}
				h, e := m.MkFamilySort(family, 0, ps[:])
				astTestMust(t, e)
				r, e := m.internal(h)
				astTestMust(t, e)
				info, e := m.sortInfo(m.sortAt(r.slot).info)
				astTestMust(t, e)
				actual, e := info.size.Size()
				astTestMust(t, e)
				if actual != want {
					t.Fatal("enumerated finite functions", left, right, rangeSize, actual, want)
				}
				for i := range ps {
					astTestMust(t, m.DelParameter(&ps[i]))
				}
			}
		}
	}
}

func TestASTArrayFiniteEveryReturnExpressionReservation(t *testing.T) {
	_, q, _ := astTestOwners(t)
	failed, completed := false, false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 28)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		family, e := m.RegisterArrayPlugin()
		astTestMust(t, e)
		domain := astArrayFiniteTestSort(t, m, 128, "return-domain")
		dp, e := m.NewParameterAST(domain)
		astTestMust(t, e)
		rp, e := m.NewParameterAST(m.external(m.boolSort))
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.MkFamilySort(family, 0, []ASTParameter{dp, rp})
		if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
			t.Fatal("exact cold return-expression graph before fault lift", ordinal, b.Used(), m.Stats())
		}
		if call != nil {
			failed = true
			if !errors.Is(call, arena.ErrInjectedFailure) {
				t.Fatal(ordinal, call)
			}
			for f := m.ArrayCardinality; f != nil; f = f.Next {
				if f.bodyCause == nil || f.sourceCause != nil || !f.sourceDone {
					t.Fatal("source return failed before reverse lexical destruction", ordinal, f)
				}
				for _, active := range f.Acquired {
					if active {
						t.Fatal("completed lexical value not destroyed", ordinal, f)
					}
				}
				if f.tokens[3] == (unumeral.OwnedRational{}) {
					t.Fatal("return expression reached before real Power acquisition", ordinal)
				}
			}
		} else {
			completed = true
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("own return-expression context end", ordinal, b.Used())
		}
		if completed {
			t.Log("first source return-expression success", ordinal)
			break
		}
	}
	if !failed || !completed {
		t.Fatal("complete actual return-expression horizon", failed, completed)
	}
}
