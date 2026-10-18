package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Seed the private CRA buffer through a completed reconstruction and its real
// result swap. No private numeral or constructor frontier is manufactured.
func craAssertionInputs(t *testing.T, m *UPolynomialManager) (*upolyVectorState, *upolyVectorState, UPolynomial) {
	t.Helper()
	a := upolySourceNew(t, m, []int32{1})
	b := upolySourceNew(t, m, []int32{2})
	ao, e := m.descriptor(a)
	serviceMust(t, e)
	bo, e := m.descriptor(b)
	serviceMust(t, e)
	p, q := unumeral.NewInt(3), unumeral.NewInt(5)
	serviceMust(t, m.combineImages(&ao.state, &p, &bo.state, &q))
	if m.cra.size != 1 || m.values(&m.cra)[0] != unumeral.NewInt(2) {
		t.Fatal("actual preceding reconstruction did not retain old C2")
	}
	serviceMust(t, m.SetInts(b, []unumeral.Int{unumeral.NewInt(2)}))
	return &ao.state, &bo.state, b
}

func TestUPolynomialCRAAssertionSourceActions(t *testing.T) {
	for _, route := range []struct {
		name                string
		b1, b2, want, bound int32
	}{
		{"first-even", 2, 3, 5, 6},
		{"second-even", 3, 2, 4, 6},
		{"noncoprime", 3, 9, 6, 27},
	} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route.name+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				c1, c2, out := craAssertionInputs(t, m)
				b1, b2 := unumeral.NewInt(route.b1), unumeral.NewInt(route.b2)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := m.combineImages(c1, &b1, c2, &b2)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("actual CRA debugger Throw", e)
					}
					if b2 != unumeral.NewInt(route.b2) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{2}) {
						t.Fatal("Throw published result or bound", b2)
					}
					if route.name == "noncoprime" {
						if m.cra.size != 0 {
							t.Fatal("gcd assertion precedes result buffer reset")
						}
					} else if m.cra.size != 1 || m.values(&m.cra)[0] != unumeral.NewInt(2) {
						t.Fatal("oddness assertion crossed result buffer reset")
					}
				} else {
					serviceMust(t, e)
					// Literal continuation centers the representative in [-bound/2,bound/2].
					want := int64(route.want)
					if want > int64(route.bound/2) {
						want -= int64(route.bound)
					}
					if b2 != unumeral.NewInt(route.bound) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{want}) {
						t.Fatal("literal CRA continuation", b2, upolyDivisionValues(t, m, out), want)
					}
				}
				if m.factorTrace != nil {
					t.Fatal("CRA debugger retained numeric borrow")
				}
			})
		}
	}
}

func TestUPolynomialCRAClosedAssertionSwitchBeforeReset(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	c1, c2, _ := craAssertionInputs(t, m)
	b1, b2 := unumeral.NewInt(3), unumeral.NewInt(5)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e := m.combineImages(c1, &b1, c2, &b2)
	if !errors.Is(e, arena.ErrClosed) || b2 != unumeral.NewInt(5) || m.cra.size != 1 || m.values(&m.cra)[0] != unumeral.NewInt(2) || (c2.size != 1 || m.values(c2)[0] != unumeral.NewInt(2)) {
		t.Fatal("closed gate changed CRA result/bound/private buffer", e)
	}
}

func TestUPolynomialCRASeparateOddnessDebuggerActions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	c1, c2, out := craAssertionInputs(t, m)
	b1, b2 := unumeral.NewInt(2), unumeral.NewInt(4)
	// First even-bound assertion continues; the independently reached second
	// assertion throws. Both precede reset and the extended-GCD locals.
	polynomialAssertionActionsFixture(t, "CT")
	e := m.combineImages(c1, &b1, c2, &b2)
	if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || b2 != unumeral.NewInt(4) || m.cra.size != 1 || m.values(&m.cra)[0] != unumeral.NewInt(2) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{2}) {
		t.Fatal("second independent CRA assertion did not Throw before reset", e)
	}
}
