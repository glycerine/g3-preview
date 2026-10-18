package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialConvertEntrySeparateSourceActions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source Throw actions erased")
	}
	for _, action := range []string{"T", "CT"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-1, 1})
			v := mustUPolyView(t, m, p)
			a, b := unumeral.NewRational(), unumeral.NewRational()
			serviceMust(t, q.SetInt64(&a, 1))
			serviceMust(t, q.SetInt64(&b, 2))
			c, d := unumeral.NewBinaryIntPower(99, 0), unumeral.NewBinaryIntPower(88, 0)
			t.Cleanup(func() {
				serviceMust(t, bqm.Del(&c))
				serviceMust(t, bqm.Del(&d))
				serviceMust(t, q.Del(&a))
				serviceMust(t, q.Del(&b))
			})
			polynomialAssertionActionsFixture(t, action)
			ok, e := m.ConvertRationalToBinaryInterval(v, q, &a, &b, bqm, &c, &d)
			if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
				t.Fatal("conversion entry Throw", ok, e)
			}
			if upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "99" || upolyIsolationBinaryRat(t, m, bqm, &d).RatString() != "88" {
				t.Fatal("entry actions crossed conversion publication")
			}
		})
	}
}
func TestUPolynomialConvertEntryContinueAndErasure(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "CC", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			choice := "CC"
			if disabled {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Direct primitive-sign adapter, not a manufactured evaluator result.
			serviceMust(t, m.convertIntervalEntryAssertions(0, 1))
		})
	}
}
func TestUPolynomialConvertEntryValidOracle(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-1, 1})
			a, b := unumeral.NewRational(), unumeral.NewRational()
			serviceMust(t, q.SetInt64(&a, 0))
			serviceMust(t, q.SetInt64(&b, 2))
			c, d := unumeral.NewBinaryRational(), unumeral.NewBinaryRational()
			t.Cleanup(func() {
				serviceMust(t, bqm.Del(&c))
				serviceMust(t, bqm.Del(&d))
				serviceMust(t, q.Del(&a))
				serviceMust(t, q.Del(&b))
			})
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &c, &d)
			serviceMust(t, e)
			if !ok || upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "0" || upolyIsolationBinaryRat(t, m, bqm, &d).RatString() != "2" {
				t.Fatal("independent dyadic conversion oracle", ok)
			}
		})
	}
}
