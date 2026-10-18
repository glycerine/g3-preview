package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialSparseCleanupPreservesIndependentLazard(t *testing.T) {
	polynomialSparseIndependentLazard(t, -1)
}
func TestPolynomialSparseIndependentLazardReservationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) { success = polynomialSparseIndependentLazard(t, prefix) })
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("sparse reservation horizon did not reach success")
	}
}
func polynomialSparseIndependentLazard(t *testing.T, prefix int64) bool {
	t.Helper()
	full := polynomialPSCFullFixture(t)
	m := full.core
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	oldInput := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	stop, reached := errors.New("independent Lazard source stop"), false
	handler, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, _ Polynomial) error {
		if c.lazardLocal.finishing && !reached {
			reached = true
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	var old polynomialLazardInvocationReceipt
	body := m.seLazardWithReceipt(3, m.unit, oldInput, 0, &oldOut.root, &old)
	if !reached || !errors.Is(body, stop) || !old.frame.active {
		t.Fatal("actual independent source stop missing", body)
	}
	before := *old.frame
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	two := unumeral.NewInt(2)
	ok, e := s.Add(&two, q)
	serviceMust(t, e)
	if !ok {
		t.Fatal("actual interpolation sample rejected")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	full.originalBudget.FailAfter(prefix)
	ok, e = s.Interpolate(out)
	full.originalBudget.FailAfter(-1)
	if e != nil && !numericPhysicalReservationRefusal(e) {
		t.Fatal("unexpected interpolation refusal", e)
	}
	success := e == nil
	if success && !ok {
		t.Fatal("actual interpolation failed")
	}
	if success && s.interpolation.active {
		t.Error("successful interpolation adopted independent pending scope")
	}
	serviceMust(t, s.CleanupInterpolation())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("interpolation cleanup consumed independent Lazard")
	}
	if success {
		polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1})
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
	serviceMust(t, out.Close())
	serviceMust(t, s.CloseChecked())
	serviceMust(t, sk.CloseChecked())
	serviceMust(t, oldOut.Close())
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{oldInput, p, q}, nil, handler.header)
	return success
}
