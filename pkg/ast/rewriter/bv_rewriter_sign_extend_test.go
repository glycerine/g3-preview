// Source: src/ast/rewriter/bv_rewriter.cpp:1525-1553.
package rewriter

import (
	"errors"
	"math/big"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOriginalSignExtend(t *testing.T) {
	m, c, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	extract, e := NewMkExtractProc(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, extract.Close()) })
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	out := astTestPin(t, m, x)
	for _, width := range []uint32{1, 8, 65, 257} {
		modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
		half := new(big.Int).Rsh(new(big.Int).Set(modulus), 1)
		for _, value := range []*big.Int{new(big.Int), new(big.Int).Sub(new(big.Int).Set(half), big.NewInt(1)), half, new(big.Int).Sub(new(big.Int).Set(modulus), big.NewInt(1))} {
			n, e := c.NumeralsField.NewValue()
			satOK(t, e)
			satOK(t, c.NumeralsField.SetString(&n, value.String()))
			input, e := m.BitVectorNumeral(&n, width)
			satOK(t, e)
			astTestPin(t, m, input)
			satOK(t, c.NumeralsField.Del(&n))
			for _, extra := range []uint32{1, 17, 65} {
				status, e := RewriteBitVectorSignExtend(extract, extra, input, false, out)
				satOK(t, e)
				got, e := m.RefGet(out)
				satOK(t, e)
				signed := new(big.Int).Set(value)
				if signed.Cmp(half) >= 0 {
					signed.Sub(signed, modulus)
				}
				want := new(big.Int).Mod(signed, new(big.Int).Lsh(big.NewInt(1), uint(width+extra)))
				n, e := c.NumeralsField.NewValue()
				satOK(t, e)
				var bits uint32
				yes, e := m.BitVectorNumeralValue(got, &n, &bits)
				satOK(t, e)
				text, e := c.NumeralsField.String(&n)
				satOK(t, e)
				satOK(t, c.NumeralsField.Del(&n))
				if status != RewriteDone || !yes || bits != width+extra || text != want.String() {
					t.Fatal("sign extension", width, extra, value, status, bits, text, want)
				}
			}
		}
	}
	satOK(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	status, e := RewriteBitVectorSignExtend(extract, 0, x, true, out)
	satOK(t, e)
	failed, e := RewriteBitVectorSignExtend(extract, 17, x, false, ast.ASTRef{})
	satOK(t, e)
	m.Budget().FailAfter(-1)
	if status != RewriteDone || failed != RewriteFailed || m.Stats() != before || !extract.cached.IsNull() {
		t.Fatal("identity/disabled elimination touched dependencies")
	}
	for _, extra := range []uint32{1, 17, 40} {
		status, e := RewriteBitVectorSignExtend(extract, extra, x, true, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		count, e := m.GetNumArgs(got)
		satOK(t, e)
		width, e := m.BitVectorSize(got)
		satOK(t, e)
		sign, e := m.GetArg(got, 0)
		satOK(t, e)
		var low, high uint32
		var base ast.ASTHandle
		yes, e := m.BitVectorExtractValue(sign, &low, &high, &base)
		satOK(t, e)
		if status != Rewrite2 || count != extra+1 || width != 8+extra || !yes || low != 7 || high != 7 || base != x {
			t.Fatal("symbolic source shape")
		}
		for i := uint32(0); i < count; i++ {
			a, e := m.GetArg(got, i)
			satOK(t, e)
			want := sign
			if i == extra {
				want = x
			}
			if a != want {
				t.Fatal("concat order", i)
			}
		}
	}
}

func TestBVRewriterSignExtendEverySymbolicReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			satOK(t, e)
			_, e = m.RegisterBitVectorPlugin()
			satOK(t, e)
			sort, e := m.BitVectorSort(8)
			satOK(t, e)
			x, e := m.MkVar(0, sort)
			satOK(t, e)
			astTestPin(t, m, x)
			out := astTestPin(t, m, x)
			var body error
			var status RewriteStatus
			extract, e := NewMkExtractProc(m)
			// Construct the caller-owned cache before arming faults.
			satOK(t, e)
			t.Cleanup(func() { satOK(t, extract.Close()) })
			m.Budget().FailAfter(ordinal)
			status, body = RewriteBitVectorSignExtend(extract, 17, x, true, out)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal("unexpected refusal", body)
				}
				got, e := m.RefGet(out)
				satOK(t, e)
				if got != x {
					t.Fatal("factory failure published output")
				}
			} else {
				success = true
				if status != Rewrite2 {
					t.Fatal("rewrite depth", status)
				}
			}
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("reservation horizon", success, failures)
	}
}
