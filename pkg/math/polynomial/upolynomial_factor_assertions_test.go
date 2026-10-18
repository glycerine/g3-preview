package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialFactorCombinationConstructorAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			ds, e := NewUPolynomialDegreeSet(fs)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, ds.Close()) })
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if action == "disabled" && polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			it, e := NewUPolynomialFactorCombination(fs, ds)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || it != nil {
					t.Fatal("source constructor Throw", it, e)
				}
			} else {
				serviceMust(t, e)
				if next, e := it.Next(false); e != nil || next {
					t.Fatal("continued empty selection", next, e)
				}
				serviceMust(t, it.Close())
			}
		})
	}
}

func TestUPolynomialFactorCombinationAssertionErasesMultiplicityRead(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	for _, c := range []int32{2, 3} {
		p := upolySourceNew(t, m, []int32{c, 1})
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
	}
	ds, e := NewUPolynomialDegreeSet(fs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, ds.Close()) })
	it, e := NewUPolynomialFactorCombination(fs, ds)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, it.Close()) })
	if next, e := it.Next(false); e != nil || !next {
		t.Fatal(next, e)
	}
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	out := upolySourceNew(t, m, []int32{97})
	// Guarded source-operand fixture only: no allocation-failure claim. The
	// multiplicity vector is not part of the continued right-product body.
	saved := fs.degrees
	fs.degrees = nil
	e = it.Right(out)
	fs.degrees = saved
	if e != nil {
		t.Fatal("disabled assertion read multiplicity", e)
	}
	sxPolyCoefficients(t, m, out, []int32{3, 1})
}

func factorAssertionPair(t *testing.T) (*UPolynomialManager, *UPolynomialFactors, *UPolynomialFactorCombination) {
	t.Helper()
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	for _, c := range []int32{2, 3} {
		p := upolySourceNew(t, m, []int32{c, 1})
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
	}
	ds, e := NewUPolynomialDegreeSet(fs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, ds.Close()) })
	it, e := NewUPolynomialFactorCombination(fs, ds)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, it.Close()) })
	return m, fs, it
}

func factorAssertionAction(t *testing.T, action string) {
	t.Helper()
	choices := action
	if action == "disabled" {
		choices = ""
	}
	polynomialAssertionActionsFixture(t, choices)
	if polynomialSourceDebug && action == "disabled" {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
}

func TestUPolynomialFactorCombinationSelectionAssertionActions(t *testing.T) {
	for _, route := range []string{"remove", "left"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, it := factorAssertionPair(t)
				out := upolySourceNew(t, m, []int32{97})
				factorAssertionAction(t, action)
				var e error
				if route == "remove" {
					_, e = it.Next(true)
				} else {
					e = it.Left(out)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("selection Throw", e)
					}
				} else if !errors.Is(e, arena.ErrBounds) || linearThrownAssertion(e) {
					t.Fatal("continued/erased selection bounds", e)
				}
				sxPolyCoefficients(t, m, out, []int32{97})
			})
		}
	}
}

func TestUPolynomialFactorCombinationRightMultiplicityAssertionPrefixes(t *testing.T) {
	for _, route := range []string{"polynomial", "tail"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, fs, it := factorAssertionPair(t)
				if next, e := it.Next(false); e != nil || !next {
					t.Fatal(next, e)
				}
				factorAssertionAction(t, action)
				// Deliberate source-precondition fixture, restored for teardown.
				// The constructor and degree-set creation used the valid factors.
				fs.degrees[1] = 2
				defer func() { fs.degrees[1] = 1 }()
				out := upolySourceNew(t, m, []int32{97})
				seed, value := unumeral.NewInt(7), unumeral.NewInt(99)
				var e error
				if route == "polynomial" {
					e = it.Right(out)
				} else {
					e = it.RightTail(&seed, &value)
				}
				thrown := polynomialSourceDebug && action == "T"
				if thrown {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("multiplicity Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if route == "polynomial" {
					want := []int32{3, 1}
					if thrown {
						want = nil
					}
					sxPolyCoefficients(t, m, out, want)
				} else {
					got, e := m.config.Integer.Int32(&value)
					serviceMust(t, e)
					want := int32(21)
					if thrown {
						want = 7
					}
					if got != want {
						t.Fatal("copy before assertion / multiplication after Continue", got, want)
					}
				}
			})
		}
	}
}

func TestUPolynomialFactorDegreeSetFinalAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			p := upolySourceNew(t, m, []int32{1, 0, 1})
			serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
			factorAssertionAction(t, action)
			// A guarded source-precondition fixture, not malloc-refusal evidence:
			// the real factor degree is two, so the subset does not contain one.
			fs.totalDegree = 1
			defer func() { fs.totalDegree = 2 }()
			ds, e := NewUPolynomialDegreeSet(fs)
			if polynomialSourceDebug && action == "T" {
				if ds != nil || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("degree-set final Throw", e)
				}
			} else {
				serviceMust(t, e)
				included, e := ds.InSet(2)
				serviceMust(t, e)
				if !included {
					t.Fatal("continued degree subset changed")
				}
				serviceMust(t, ds.Close())
			}
		})
	}
}
