// Source: src/ast/bv_decl_plugin.cpp:169-178.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTBitVectorOriginalFamilySortDispatch(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, width := range []int32{1, 8, 65, 257} {
		p, e := m.NewParameterInt(width)
		astTestMust(t, e)
		h, e := m.MkFamilySort(fid, 0, []ASTParameter{p})
		astTestMust(t, e)
		want, e := m.BitVectorSort(uint32(width))
		astTestMust(t, e)
		if h != want {
			t.Fatal("source sort cache identity", width, h, want)
		}
		// The BV plugin's sort factory ignores the supplied declaration kind.
		m.Budget().FailAfter(0)
		again, e := m.MkFamilySort(fid, 999, []ASTParameter{p})
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if again != h {
			t.Fatal("source ignored kind", width)
		}
		astTestMust(t, m.DelParameter(&p))
	}
}

func TestASTBitVectorOriginalFamilySortGuards(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, ps := range [][]ASTParameter{nil, {{}, {}}} {
		_, e := m.MkFamilySort(fid, 0, ps)
		if e == nil || e.Error() != "expecting one integer parameter to bit-vector sort" {
			t.Fatal("count guard erases parameter reads", e)
		}
	}
	symbol, e := m.NewParameterSymbol(astTestSymbol(t, m, "bad-bv-sort-width"))
	astTestMust(t, e)
	_, e = m.MkFamilySort(fid, 0, []ASTParameter{symbol})
	if e == nil || e.Error() != "expecting one integer parameter to bit-vector sort" {
		t.Fatal("kind guard", e)
	}
	astTestMust(t, m.DelParameter(&symbol))
	p, e := m.NewParameterInt(0)
	astTestMust(t, e)
	_, e = m.MkFamilySort(fid, 0, []ASTParameter{p})
	if e == nil || e.Error() != "bit-vector size must be greater than zero" {
		t.Fatal("zero guard", e)
	}
	astTestMust(t, m.DelParameter(&p))
}

func TestASTBitVectorFamilySortEveryColdReservationAndFinalize(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		func() {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			p, e := m.NewParameterInt(65)
			astTestMust(t, e)
			m.Budget().FailAfter(ordinal)
			_, body := m.MkFamilySort(fid, 0, []ASTParameter{p})
			m.Budget().FailAfter(-1)
			if body != nil && !errors.Is(body, arena.ErrInjectedFailure) {
				t.Fatal(ordinal, body)
			}
			if body == nil {
				success = true
			}
			astTestMust(t, m.DelParameter(&p))
			astTestMust(t, m.SourceFinalize())
			if m.bitVectorPlugin != nil {
				t.Fatal("family sort source cache owner retained")
			}
		}()
		if success {
			return
		}
	}
	t.Fatal("unreached cold family sort")
}

func TestASTBitVectorFamilySortUsesExplicitCacheBeyondGetterCutoff(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterInt(4096)
	astTestMust(t, e)
	h, e := m.MkFamilySort(fid, 0, []ASTParameter{p})
	astTestMust(t, e)
	if len(m.bitVectorPlugin.sorts) != 4097 || m.bitVectorPlugin.sortAcquired[4096] != 1 {
		t.Fatal("mk_sort must explicitly cache large widths")
	}
	r, e := m.bitVectorPlugin.getSort(4096)
	astTestMust(t, e)
	if m.external(r) != h || m.nodeAt(r.slot).rc != 1 {
		t.Fatal("large getter must reuse identity without another cache INC")
	}
	astTestMust(t, m.DelParameter(&p))
}
