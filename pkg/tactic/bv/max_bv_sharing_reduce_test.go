package bv

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func sharingMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func sharingFixture(t *testing.T) (*ast.ASTManager, ast.FamilyID) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, e := symbol.NewManager(symbol.Config{Budget: b})
	sharingMust(t, e)
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	sharingMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	sharingMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	sharingMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	sharingMust(t, e)
	t.Cleanup(func() {
		sharingMust(t, m.Close())
		sharingMust(t, q.Close())
		sharingMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("owner roots", b.Used())
		}
	})
	return m, fid
}

func TestSharingReducerOrderedPairsReuseAndNumeralPosition(t *testing.T) {
	m, fid := sharingFixture(t)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(900+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	num, e := m.BitVectorNumeralUnsigned(3, 8)
	sharingMust(t, e)
	sharingMust(t, m.IncRef(num))
	t.Cleanup(func() { sharingMust(t, m.DecRef(num)) })
	for _, kind := range []int32{4, 6, 31, 33} {
		binary, e := m.MkFamilyApp(fid, kind, nil, vars[:2])
		sharingMust(t, e)
		sharingMust(t, m.IncRef(binary))
		decl, e := m.GetDecl(binary)
		sharingMust(t, e)
		r, e := NewSharingReducer(m, fid, 128)
		sharingMust(t, e)
		_, changed, e := r.ReduceApp(decl, vars[:2])
		sharingMust(t, e)
		if changed {
			t.Fatal("binary source branch must register and return BR_FAILED", kind)
		}
		// The first pair is nonadjacent. Greedy reuse brings it together before
		// the tree grouping, preserving its source order even when reversed.
		got, changed, e := r.ReduceApp(decl, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
		sharingMust(t, e)
		if !changed {
			t.Fatal("n-ary BV app was not reduced", kind)
		}
		a, e := m.GetArg(got, 0)
		sharingMust(t, e)
		a, e = m.GetArg(a, 0)
		sharingMust(t, e)
		if a != binary {
			t.Fatal("did not reuse ordered registered pair", kind, a, binary)
		}
		gotWithNum, changed, e := r.ReduceApp(decl, []ast.ASTHandle{num, vars[0], vars[1]})
		sharingMust(t, e)
		if !changed {
			t.Fatal("numeral n-ary reduction", kind)
		}
		first, e := m.GetArg(gotWithNum, 0)
		sharingMust(t, e)
		if first != num {
			t.Fatal("first numeral lost its source position", kind)
		}
		sharingMust(t, r.Close())
		sharingMust(t, m.DecRef(binary))
	}
}
