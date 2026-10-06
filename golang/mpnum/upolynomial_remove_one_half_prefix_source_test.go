// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Literal pseudo-division and independent exact scaled quotient cases.
package mpnum

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUPolynomialRemoveOneHalfRootEveryActualPrefix(t *testing.T) {
	success, acquired := false, false
	for prefix := int64(0); prefix < 160 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			var big, twice, neg Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			two := NewInt(2)
			serviceMust(t, z.Mul(&big, &two, &twice))
			serviceMust(t, z.Copy(&neg, &big))
			serviceMust(t, z.Neg(&neg))
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []Int{neg, twice}))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, z.Del(&big))
			serviceMust(t, z.Del(&twice))
			serviceMust(t, z.Del(&neg))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			base := z.budget.Used() - integerActualOwnedBytes(z) - p.oneHalfRootHistoryBytes - p.removeOneHalfRootHistoryBytes - removeHalfVectorCold(p)
			z.budget.FailAfter(prefix)
			body := p.RemoveOneHalfRoot(v, out)
			q := &p.removeOneHalfRoot
			r := p.removeOneHalfRootReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*Int{&q.values[0], &p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}
			if z.matchesAddMulInvocation(&q.addMul) {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			for _, child := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
				if z.matchesRawInvocation(child) {
					for i := uint8(0); i < child.frame.count; i++ {
						extra = append(extra, &child.frame.values[i])
					}
				}
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{input, out}, extra, nil)
			if got, want := z.budget.Used(), base+integerActualOwnedBytes(z)+p.oneHalfRootHistoryBytes+p.removeOneHalfRootHistoryBytes+removeHalfVectorCold(p); got != want {
				t.Fatal("before-lift cold graph", got, want)
			}
			if polynomialNegNumericPending(&q.numeric) || z.matchesAddMulInvocation(&q.addMul) {
				acquired = true
			}
			z.budget.FailAfter(-1)
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
				t.Fatalf("later source failure=%v prior=%v failedRow=%v/%d/%d/%d/%v tmp=%v div1=%v div2=%v", laterError, body, q.failedRowStore, q.failedRowBacking, q.failedRowGeneration, q.failedRowIndex, q.failedRowCell, z.tmp, p.div1, p.div2)
			}
			current := p.removeOneHalfRoot
			used, physical := z.budget.Used(), z.memory.Stats()
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			if !reflect.DeepEqual(current, p.removeOneHalfRoot) || z.budget.Used() != used || z.memory.Stats() != physical || (r.history != nil && *r.history != old) {
				t.Fatal("old ACK acquired later frame")
			}
			prior := z.removeOneHalfRootCaller
			z.removeOneHalfRootCaller = p
			if context.roots(z) != nil {
				t.Fatal("stale context acquired reused caller")
			}
			z.removeOneHalfRootCaller = prior
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
				if *a != NewInt(n) {
					t.Fatal("pseudo-division coefficient", i, *a, n)
				}
			}
		})
	}
}

// Actual directory/backing capacities, independent of row SIZE and NUM census.
func removeHalfVectorCold(p *UPolynomialManager) uint64 {
	n := uint64(cap(p.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size()) + uint64(cap(p.store.backings))*uint64(reflect.TypeOf(upolyBacking{}).Size())
	for _, b := range p.store.backings {
		if b.live {
			n += uint64(cap(b.values)) * uint64(reflect.TypeOf(Int{}).Size())
		}
	}
	return n
}
