// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Literal pseudo-division and independent exact scaled quotient cases.
package polynomial

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialRemoveOneHalfRootEveryActualPrefix(t *testing.T) {
	success, acquired := false, false
	for prefix := int64(0); prefix < 160 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			var big, twice, neg util.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			two := util.NewInt(2)
			serviceMust(t, z.Mul(&big, &two, &twice))
			serviceMust(t, z.Copy(&neg, &big))
			serviceMust(t, z.Neg(&neg))
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []util.Int{neg, twice}))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, z.Del(&big))
			serviceMust(t, z.Del(&twice))
			serviceMust(t, z.Del(&neg))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			base := z.Budget().Used() - integerActualOwnedBytes(z) - p.oneHalfRootHistoryBytes - p.removeOneHalfRootHistoryBytes
			z.Budget().FailAfter(prefix)
			body := p.RemoveOneHalfRoot(v, out)
			q := &p.removeOneHalfRoot
			r := p.removeOneHalfRootReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*util.Int{&q.values[0], &p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}
			if z.MatchesAddMulInvocation(&q.addMul) {
				extra = append(extra, &q.addMul.Frame.Tmp)
			}
			for _, child := range []*util.IntegerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
				if z.MatchesRawInvocation(child) {
					for i := uint8(0); i < child.Frame.Count; i++ {
						extra = append(extra, &child.Frame.Values[i])
					}
				}
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{input, out}, extra, nil)
			if got, want := z.Budget().Used(), base+integerActualOwnedBytes(z)+p.oneHalfRootHistoryBytes+p.removeOneHalfRootHistoryBytes; got != want {
				t.Fatal("before-lift cold graph", got, want)
			}
			if polynomialNegNumericPending(&q.numeric) || z.MatchesAddMulInvocation(&q.addMul) {
				acquired = true
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.DelPolynomial(&input))
			serviceMust(t, p.DelPolynomial(&out))
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			var old upolyRemoveOneHalfRootHistory
			if r.history != nil {
				old = *r.history
			}
			small := algebraicPolynomial(t, p, -1, 2)
			smallV, e := p.wholeView(small)
			serviceMust(t, e)
			later, e := p.NewPolynomial()
			serviceMust(t, e)
			laterError := p.RemoveOneHalfRoot(smallV, later)
			if laterError != nil {
				t.Fatalf("later source failure=%v prior=%v failedRow=%v/%d/%d/%d/%v tmp=%v div1=%v div2=%v", laterError, body, q.failedRowStore, q.failedRowBacking, q.failedRowGeneration, q.failedRowIndex, q.failedRowCell, z.Tmp, p.div1, p.div2)
			}
			current := p.removeOneHalfRoot
			used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			if !reflect.DeepEqual(current, p.removeOneHalfRoot) || z.Budget().Used() != used || z.RuntimeOwner().Stats() != physical || (r.history != nil && *r.history != old) {
				t.Fatal("old ACK acquired later frame")
			}
			prior := removeOneHalfRootCaller(z)
			setRemoveOneHalfRootCaller(z, p)
			if (context.roots(z)) != nil {
				t.Fatal("stale context acquired reused caller")
			}
			setRemoveOneHalfRootCaller(z, prior)
			serviceMust(t, p.DelPolynomial(&later))
			serviceMust(t, p.CleanupRemoveOneHalfRoot())
			success = body == nil
		})
	}
	if !success || !acquired {
		t.Fatal("pending/success frontier absent", success, acquired)
	}
}

func TestUPolynomialRemoveOneHalfRootIndependentLiteralCases(t *testing.T) {
	cases := []struct{ input, want []int32 }{{[]int32{-1, 2}, []int32{2}}, {[]int32{1, -4, 4}, []int32{-4, 8}}, {[]int32{-1, 1, 2}, []int32{4, 4}}}
	for _, cse := range cases {
		t.Run(fmt.Sprint(cse.input), func(t *testing.T) {
			_, p := algebraicFixture(t)
			input := algebraicPolynomial(t, p, cse.input...)
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			defer func() { serviceMust(t, p.DelPolynomial(&out)) }()
			v, e := p.wholeView(input)
			serviceMust(t, e)
			serviceMust(t, p.RemoveOneHalfRoot(v, out))
			ov, e := p.wholeView(out)
			serviceMust(t, e)
			if ov.count != uint32(len(cse.want)) {
				t.Fatal("degree", ov.count)
			}
			for i, n := range cse.want {
				a, e := p.read(ov, uint32(i))
				serviceMust(t, e)
				if *a != util.NewInt(n) {
					t.Fatal("pseudo-division coefficient", i, *a, n)
				}
			}
		})
	}
}
