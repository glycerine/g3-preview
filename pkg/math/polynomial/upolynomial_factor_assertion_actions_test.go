package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialFactorMultiplicityAssertionActions(t *testing.T) {
	for _, move := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(map[bool]string{false: "copy", true: "move"}[move]+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				fs, e := m.NewFactors()
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, fs.Close()) })
				p, out := upolySourceNew(t, m, []int32{2, 3, 1}), upolySourceNew(t, m, []int32{99})
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				if move {
					e = fs.PushMove(p, 0)
				} else {
					e = fs.PushCopy(mustUPolyView(t, m, p), 0)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || fs.count != 0 || fs.totalFactors != 0 || fs.totalDegree != 0 {
						t.Fatal("multiplicity Throw mutated factor", e)
					}
					upolyGCDExpect(t, m, p, []int64{2, 3, 1})
					return
				}
				serviceMust(t, e)
				if fs.count != 1 || fs.degrees[0] != 0 || fs.totalFactors != 0 || fs.totalDegree != 0 {
					t.Fatal("zero-degree Continue counters", fs.count, fs.totalFactors, fs.totalDegree)
				}
				if move {
					upolyGCDExpect(t, m, p, nil)
				} else {
					upolyGCDExpect(t, m, p, []int64{2, 3, 1})
				}
				serviceMust(t, fs.Multiply(out))
				// Source multiply's degree>1 test still multiplies this degree0 factor once.
				upolyGCDExpect(t, m, out, []int64{2, 3, 1})
			})
		}
	}
}
func TestUPolynomialFactorZeroConstantAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			p, out := upolySourceNew(t, m, []int32{1, 1}), upolySourceNew(t, m, []int32{99})
			serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
			zero := unumeral.NewInt(0)
			serviceMust(t, fs.SetConstant(&zero))
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = fs.Multiply(out)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("zero-constant Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			// reset(out) precedes even a Throw at the zero-constant assertion.
			upolyGCDExpect(t, m, out, nil)
			if fs.count != 1 || fs.totalFactors != 1 || fs.totalDegree != 1 {
				t.Fatal("zero-constant branch changed factor collection")
			}
		})
	}
}
func TestUPolynomialFactorClosedAssertionGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	for _, move := range []bool{false, true} {
		t.Run(map[bool]string{false: "copy", true: "move"}[move], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			p := upolySourceNew(t, m, []int32{1, 1})
			descriptor, e := m.descriptor(p)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			if move {
				e = fs.PushMove(p, 1)
			} else {
				e = fs.PushCopy(mustUPolyView(t, m, p), 1)
			}
			if !errors.Is(e, arena.ErrClosed) || fs.count != 0 {
				t.Fatal("closed runtime crossed factor insertion", e)
			}
			if descriptor.state.size != 2 || m.values(&descriptor.state)[0] != unumeral.NewInt(1) || m.values(&descriptor.state)[1] != unumeral.NewInt(1) {
				t.Fatal("closed gate changed actual input backing")
			}
		})
	}
}
