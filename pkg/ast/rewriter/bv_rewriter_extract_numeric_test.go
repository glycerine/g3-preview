package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:590-790.

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"math/big"
	"testing"
)

func TestBVRewriterExtractEveryActualNumericReservation(t *testing.T) {
	for _, tc := range []struct {
		text             string
		width, high, low uint32
	}{
		{"-128", 8, 7, 4},
		{"73786976294838206467", 257, 65, 1},
		{"-3", 257, 256, 64},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				value, _ := new(big.Int).SetString(tc.text, 10)
				arg := bvOrRawNumeral(t, m, q, fid, tc.width, value)
				p := bvNotTestOwner(t, m, false, false)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteExtract(tc.high, tc.low, arg, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					_, replay := p.RewriteExtract(tc.high, tc.low, arg, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					if st != RewriteDone || p.extraction != nil {
						t.Fatal("source return", st)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					n, e := q.NewValue()
					astTestMust(t, e)
					var bits uint32
					_, e = m.BitVectorNumeralValue(got, &n, &bits)
					astTestMust(t, e)
					text, e := q.String(&n)
					astTestMust(t, e)
					want := new(big.Int).Set(value)
					width := tc.high - tc.low + 1
					modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
					if want.Sign() < 0 {
						want.Mod(want, modulus)
					}
					want.Rsh(want, uint(tc.low))
					want.Mod(want, modulus)
					if text != want.String() || bits != width {
						t.Fatal("independent source numeric oracle", text, want, bits, width)
					}
					astTestMust(t, q.Del(&n))
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete || failures == 0 {
			t.Fatal("campaign incomplete", tc, complete, failures)
		}
	}
}

func TestBVRewriterExtractPropagationEveryActualNumericReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, q, fid := bvOrRuntimeFixture(t)
			value := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 70), big.NewInt(3))
			prefix := bvOrRawNumeral(t, m, q, fid, 65, value)
			x := bvPropagationVar(t, m, 0, 8)
			p := bvNotTestOwner(t, m, false, false)
			p.polynomial.config.bitVector.ExtractPropagation = true
			cat := bvPropagationConcat(t, m, prefix, x)
			arg := bvPropagationOp(t, m, p, 4, cat, cat)
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteExtract(70, 1, arg, out)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				_, replay := p.RewriteExtract(70, 1, arg, out)
				if replay != body {
					t.Fatal("stopped replay", replay, body)
				}
			} else {
				complete = true
				if st != Rewrite2 || p.extraction != nil || p.propagation != nil {
					t.Fatal("source return", st)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				hi, e := m.BitVectorExtractHigh(got)
				astTestMust(t, e)
				lo, e := m.BitVectorExtractLow(got)
				astTestMust(t, e)
				sum, e := m.GetArg(got, 0)
				astTestMust(t, e)
				count, e := m.GetNumArgs(sum)
				astTestMust(t, e)
				if hi != 70 || lo != 1 || count != 2 {
					t.Fatal("propagated extract shape", hi, lo, count)
				}
				for i := uint32(0); i < count; i++ {
					c, e := m.GetArg(sum, i)
					astTestMust(t, e)
					head, e := m.GetArg(c, 0)
					astTestMust(t, e)
					tail, e := m.GetArg(c, 1)
					astTestMust(t, e)
					n, e := q.NewValue()
					astTestMust(t, e)
					var bits uint32
					_, e = m.BitVectorNumeralValue(head, &n, &bits)
					astTestMust(t, e)
					text, e := q.String(&n)
					astTestMust(t, e)
					want := new(big.Int).Mod(new(big.Int).Set(value), new(big.Int).Lsh(big.NewInt(1), 63))
					if bits != 63 || text != want.String() || tail != x {
						t.Fatal("source shortening", text, want, bits)
					}
					astTestMust(t, q.Del(&n))
				}
			}
			cause := p.cause
			astTestMust(t, p.Close())
			astTestMust(t, p.Close())
			if p.cause != cause {
				t.Fatal("history changed")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("actual propagation incomplete", complete, failures)
	}
}
