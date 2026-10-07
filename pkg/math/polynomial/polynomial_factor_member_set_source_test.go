// Source: src/math/polynomial/polynomial.cpp:1613-1642.
package polynomial

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactorResetSETOneActualInvocation(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.CloseChecked()) }()
	prior := fs.memberSet.invocation
	serviceMust(t, fs.Reset())
	if fs.memberSet.invocation <= prior || fs.memberSet.executing || fs.memberSet.active || fs.memberSet.numeric.Cell.Object != (full.core.integer.PendingAllocationObject()) {
		t.Fatal("reached class member SET needs its own concrete completed invocation")
	}
	if fs.constant != util.NewInt(1) {
		t.Fatal("literal reset SET one")
	}
	polynomialFactorsAudit(t, fs, nil, nil)
}

func TestPolynomialFactorResetSETOneEveryFieldReservationPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 96 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			z := full.core.integer
			fs, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			var modulus util.Int
			one := util.NewInt(1)
			serviceMust(t, z.Mul2k(&one, 521, &modulus))
			serviceMust(t, z.Sub(&modulus, &one, &modulus))
			serviceMust(t, full.core.modular.SetZP(&modulus))
			oldInvocation := fs.memberSet.invocation
			z.Budget().FailAfter(prefix)
			cause := fs.Reset()
			polynomialFactorsAudit(t, fs, nil, []*util.Int{&modulus})
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual field SET1 reservation", cause)
			}
			if cause == nil {
				success = true
				if fs.constant != util.NewInt(1) {
					t.Fatal("literal reset value")
				}
			}
			if fs.memberSet.invocation > oldInvocation && fs.memberSet.numeric.Cell.Object != z.PendingAllocationObject() {
				t.Fatal("actual reached SET1 descriptor authority")
			}
			z.Budget().FailAfter(-1)
			body, source := fs.memberSet.bodyCause, fs.memberSet.sourceCause
			serviceMust(t, fs.CloseChecked())
			if fs.memberSet.bodyCause != body || fs.memberSet.sourceCause != source {
				t.Fatal("own acknowledgment erased member SET history")
			}
			serviceMust(t, z.Del(&modulus))
		})
	}
	if !success {
		t.Fatal("actual first successful field SET1 never reached")
	}
}
