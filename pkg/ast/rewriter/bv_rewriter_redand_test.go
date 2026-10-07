// Source: src/ast/rewriter/bv_rewriter.cpp:2048-2056.
package rewriter

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOriginalReductionAND(t *testing.T) {
	m, c, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	bit0, e := m.BitVectorNumeralUnsigned(0, 1)
	satOK(t, e)
	astTestPin(t, m, bit0)
	bit1, e := m.BitVectorNumeralUnsigned(1, 1)
	satOK(t, e)
	astTestPin(t, m, bit1)
	out := astTestPin(t, m, bit0)
	for _, width := range []uint32{1, 8, 65, 257} {
		all := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
		for _, value := range []*big.Int{new(big.Int), big.NewInt(1), all, new(big.Int).Sub(new(big.Int).Set(all), big.NewInt(1))} {
			n, e := c.NumeralsField.NewValue()
			satOK(t, e)
			satOK(t, c.NumeralsField.SetString(&n, value.String()))
			input, e := m.BitVectorNumeral(&n, width)
			satOK(t, e)
			astTestPin(t, m, input)
			satOK(t, c.NumeralsField.Del(&n))
			status, e := RewriteBitVectorReductionAND(m, input, out)
			satOK(t, e)
			got, e := m.RefGet(out)
			satOK(t, e)
			want := bit0
			if value.Cmp(all) == 0 {
				want = bit1
			}
			if status != RewriteDone || got != want {
				t.Fatal("reduction AND", width, value, status, got, want)
			}
		}
	}
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	satOK(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	status, e := RewriteBitVectorReductionAND(m, x, out)
	satOK(t, e)
	emptyStatus, e := RewriteBitVectorReductionAND(m, x, ast.ASTRef{})
	satOK(t, e)
	m.Budget().FailAfter(-1)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != RewriteFailed || emptyStatus != RewriteFailed || got != x || m.Stats() != before {
		t.Fatal("unmatched reduction allocated/read output")
	}
}
