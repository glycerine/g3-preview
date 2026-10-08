// Source: src/ast/rewriter/bv_rewriter.cpp:1525-1553.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"math/big"
	"testing"
)

func TestBVRewriterSignExtensionEveryActualNumericReservation(t *testing.T) {
	for _, tc := range []struct {
		text          string
		width, amount uint32
	}{
		{"36893488147419103231", 65, 17},
		{"-3", 257, 65},
		{"73786976294838206467", 65, 17},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				value, _ := new(big.Int).SetString(tc.text, 10)
				arg := bvOrRawNumeral(t, m, q, fid, tc.width, value)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteSignExtend(tc.amount, arg, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					_, replay := p.RewriteSignExtend(tc.amount, arg, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					if st != RewriteDone || p.signExtension != nil {
						t.Fatal("source return", st)
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					v, e := q.NewValue()
					astTestMust(t, e)
					var bits uint32
					_, e = m.BitVectorNumeralValue(got, &v, &bits)
					astTestMust(t, e)
					text, e := q.String(&v)
					astTestMust(t, e)
					astTestMust(t, q.Del(&v))
					modulus := new(big.Int).Lsh(big.NewInt(1), uint(tc.width))
					want := new(big.Int).Mod(new(big.Int).Set(value), modulus)
					if want.Bit(int(tc.width-1)) != 0 {
						want.Sub(want, modulus)
					}
					want.Mod(want, new(big.Int).Lsh(big.NewInt(1), uint(tc.width+tc.amount)))
					if text != want.String() || bits != tc.width+tc.amount {
						t.Fatal("independent signed-extension oracle", text, want, bits)
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
			t.Fatal("actual extension incomplete", tc, complete, failures)
		}
	}
}
