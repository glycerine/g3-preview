// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Real BIG finite-field Div/Inv/PN/RAW children; no replacement division engine.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialRemoveOneHalfRootFieldEveryActualPrefix(t *testing.T) {
	success, child := false, false
	for prefix := int64(0); prefix < 180 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			digits := make([]uint32, 17)
			for i := range digits {
				digits[i] = ^uint32(0)
			}
			digits[16] = 511
			var prime, b, twice, neg util.Int
			serviceMust(t, z.SetLimbs(&prime, digits))
			serviceMust(t, p.SetZP(&prime))
			digits[16] = 255
			serviceMust(t, z.SetLimbs(&b, digits))
			two := util.NewInt(2)
			serviceMust(t, z.Mul(&b, &two, &twice))
			serviceMust(t, z.Copy(&neg, &b))
			serviceMust(t, z.Neg(&neg))
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []util.Int{neg, twice}))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, z.Del(&twice))
			serviceMust(t, z.Del(&neg))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			base := z.Budget().Used() - integerActualOwnedBytes(z) - p.oneHalfRootHistoryBytes - p.removeOneHalfRootHistoryBytes
			z.Budget().FailAfter(prefix)
			body := p.RemoveOneHalfRoot(v, out)
			q := &p.removeOneHalfRoot
			r := p.removeOneHalfRootReceipt()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*util.Int{&prime, &b, &q.values[0], &p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}
			if z.MatchesAddMulInvocation(&q.addMul) {
				extra = append(extra, &q.addMul.Frame.Tmp)
			}
			for _, raw := range []*util.IntegerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
				if z.MatchesRawInvocation(raw) {
					for i := uint8(0); i < raw.Frame.Count; i++ {
						extra = append(extra, &raw.Frame.Values[i])
					}
				}
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{input, out}, extra, nil)
			if got, want := z.Budget().Used(), base+integerActualOwnedBytes(z)+p.oneHalfRootHistoryBytes+p.removeOneHalfRootHistoryBytes; got != want {
				t.Fatal("field before-lift cold graph", got, want)
			}
			if q.normalize.Invocation != 0 || q.extendedRaw.Invocation != 0 || q.divisionRaw.Invocation != 0 {
				child = true
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
			if body == nil {
				ov, e := p.wholeView(out)
				serviceMust(t, e)
				if ov.count != 1 {
					t.Fatal("field quotient degree", ov.count)
				}
				a, e := p.read(ov, 0)
				serviceMust(t, e)
				equal, e := z.Equal(a, &b)
				serviceMust(t, e)
				if !equal {
					t.Fatal("field independent quotient")
				}
				success = true
			}
			serviceMust(t, p.DelPolynomial(&input))
			serviceMust(t, p.DelPolynomial(&out))
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			serviceMust(t, p.SetZ())
			serviceMust(t, z.Del(&prime))
			serviceMust(t, z.Del(&b))
		})
	}
	if !success || !child {
		t.Fatal("actual field child/success absent", success, child)
	}
}
