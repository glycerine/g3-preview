package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

// (x-B)*(x-(B+3)), B=2^64+1, bracket (B-1/4,B+1/2).
// B sits one third of the way through this interval, so dyadic bisection never
// discovers it exactly. IsInt must use comparison, integer Horner and SET.
func algebraicIntegerPredicateRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z, b := c.config.Polynomials, c.config.Rationals.Integers(), c.config.Binary
	var coefficients [3]unumeral.Int
	for i, text := range []string{"340282366920938463555608327800315969540", "-36893488147419103237", "1"} {
		serviceMust(t, z.SetString(&coefficients[i], text))
	}
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, coefficients[:]))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	var n unumeral.Int
	serviceMust(t, z.SetString(&n, "73786976294838206467"))
	serviceMust(t, b.SetIntegerPower(&lower, &n, 2))
	serviceMust(t, z.SetString(&n, "36893488147419103235"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 1))
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, v, &lower, &upper, false))
	serviceMust(t, z.Del(&n))
	serviceMust(t, b.Del(&lower))
	serviceMust(t, b.Del(&upper))
	serviceMust(t, p.DelPolynomial(&poly))
	for i := range coefficients {
		serviceMust(t, z.Del(&coefficients[i]))
	}
	return root
}
func algebraicPredicateBeforeIntegerEvaluation(t *testing.T, c *AlgebraicRootCore, root *Algebraic) UPolynomialView {
	t.Helper()
	ok, e := c.refineAt(root, 1, true)
	serviceMust(t, e)
	if !ok {
		t.Fatal("integer fixture discovered by dyadic refinement")
	}
	l, e := c.location(root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.floorPredicateCandidate(r.interval.BorrowUpper()))
	less, e := c.lessPredicateInteger(r.interval.BorrowLower())
	serviceMust(t, e)
	if !less {
		t.Fatal("fixture candidate is outside interval")
	}
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	return v
}
func TestAlgebraicPredicateIntegerPublicSetterSequence(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	root := algebraicIntegerPredicateRoot(t, c)
	yes, e := c.IsInt(&root)
	serviceMust(t, e)
	if !yes || c.integerSet.invocation != 1 || c.integerSet.sourceCursor != 2 || c.integerSet.active || c.localActive || c.predicate.active {
		t.Fatal("integer predicate bypassed source wrapper SET")
	}
	want := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&want, "18446744073709551617"))
	cmp, e := c.CompareRational(&root, &want)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("independent integer candidate", cmp)
	}
	serviceMust(t, c.config.Rationals.Del(&want))
	serviceMust(t, c.Del(&root))
}
func TestAlgebraicPredicateIntegerSetterAllocationPrefixes(t *testing.T) {
	success, issued, refused := false, false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _, _ := algebraicNegTLSFixture(t, nil)
			root := algebraicIntegerPredicateRoot(t, c)
			v := algebraicPredicateBeforeIntegerEvaluation(t, c, &root)
			sign, e := c.evalPredicateInteger(v)
			serviceMust(t, e)
			if sign != 0 {
				t.Fatal("integer candidate is not root")
			}
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(ordinal)
			body := c.setIntegerWithInvocation(&root, &c.predicate.candidate, &c.predicate.setter)
			z.Budget().FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
			} else {
				success = true
			}
			if !c.matchesIntegerSetter(&c.predicate.setter) {
				t.Fatal("accepted integer setter receipt absent")
			}
			if c.integerSet.numeratorCopy != (unumeral.IntegerCopyAllocationReceipt{}) {
				issued = true
			}
			if c.integerSet.active {
				c.predicate.childStop = c.integerSet.sourceStop
			}
			_ = c.finishPredicate(body)
			serviceMust(t, c.CleanupRetained())
			if c.integerSet.active || c.predicate.active {
				t.Fatal("setter owner did not finish")
			}
			if success {
				want := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&want, "18446744073709551617"))
				cmp, e := c.CompareRational(&root, &want)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("setter changed integer candidate")
				}
				serviceMust(t, c.config.Rationals.Del(&want))
			}
			serviceMust(t, c.Del(&root))
		})
	}
	if !success || !issued || !refused {
		t.Fatal("missing actual setter allocation prefixes", success, issued, refused)
	}
}
func TestAlgebraicPredicateIntegerHornerDoesNotAdoptOlderInvocation(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicIntegerPredicateRoot(t, c)
	v := algebraicPredicateBeforeIntegerEvaluation(t, c, &root)
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignInt(v, &c.predicate.candidate)
	if body == nil || !p.integerEvaluation.active {
		t.Fatal("older integer Horner did not retain allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	_, body = c.evalPredicateInteger(v)
	if body == nil || c.predicate.integerEvaluation.owner != nil || c.predicate.childStop != nil {
		t.Fatal("refused integer evaluation acquired older scope", body)
	}
	prior := p.integerEvaluation
	_ = c.finishPredicate(body)
	serviceMust(t, c.CleanupRetained())
	if !reflect.DeepEqual(prior, p.integerEvaluation) {
		t.Fatal("predicate cleanup consumed older integer Horner")
	}
	serviceMust(t, p.CleanupIntegerEvaluation())
	serviceMust(t, c.Del(&root))
	if e := c.cleanupPredicate(); errors.Is(e, ErrAlgebraicPending) {
		t.Fatal(e)
	}
}

func TestAlgebraicPredicateIntegerComparisonActualCopy(t *testing.T) {
	var c *AlgebraicRootCore
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.predicate.arithmeticExecuting {
			return f
		}
		reached = true
		for _, e := range []error{c.Close(), c.CleanupRetained(), c.finishPredicate(nil)} {
			if !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("integer comparison callback obtained lifecycle", e)
			}
		}
		return f
	}
	var runtime *umemory.Runtime
	c, _, runtime = algebraicNegTLSFixture(t, transport)
	z, b := c.config.Rationals.Integers(), c.config.Binary
	var n unumeral.Int
	var upper, lower unumeral.BinaryRational
	serviceMust(t, z.SetString(&n, "36893488147419103235"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 1))
	serviceMust(t, z.SetString(&n, "73786976294838206467"))
	serviceMust(t, b.SetIntegerPower(&lower, &n, 2))
	serviceMust(t, z.Del(&n))
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.floorPredicateCandidate(&upper))
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := c.lessPredicateInteger(&lower)
	s := &c.predicate
	if body == nil || !reached || !z.MatchesCopyAllocationInvocation(&s.integerComparison) {
		t.Fatal("actual integer comparison copy not reached", body, reached)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	if _, e := c.lessPredicateInteger(&lower); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("outstanding integer comparison replaced", e)
	}
	_ = c.finishPredicate(body)
	serviceMust(t, c.CleanupRetained())
	if s.active || z.MatchesCopyAllocationInvocation(&s.integerComparison) {
		t.Fatal("integer comparison copy unacknowledged")
	}
	serviceMust(t, b.Del(&lower))
	serviceMust(t, b.Del(&upper))
}

func TestAlgebraicPredicateIntegerHornerErasedCallsIssueNoReceipt(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	constant, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(constant, []unumeral.Int{unumeral.NewInt(-7)}))
	v, e := p.wholeView(constant)
	serviceMust(t, e)
	for _, tc := range []struct {
		v    UPolynomialView
		want int
	}{{UPolynomialView{}, 0}, {v, -1}} {
		var receipt upolyIntegerEvaluationInvocationReceipt
		got, e := p.evalSignIntegerWithReceipt(tc.v, nil, &receipt)
		serviceMust(t, e)
		if got != tc.want || receipt.owner != nil {
			t.Fatal("erased evaluation issued integer Horner receipt", got)
		}
	}
	serviceMust(t, p.DelPolynomial(&constant))
	if c.predicate.active {
		t.Fatal("erased Horner entered predicate")
	}
}

func TestAlgebraicPredicateIntegerSetterDoesNotAdoptOlderInvocation(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	z := c.config.Rationals.Integers()
	var out Algebraic
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.SetInteger(&out, &z.Two64)
	if body == nil || !c.integerSet.active || !z.MatchesCopyAllocationInvocation(&c.integerSet.numeratorCopy) {
		t.Fatal("older setter did not retain actual NUM copy", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, c.beginPredicate())
	upper := unumeral.NewBinaryInt(2)
	serviceMust(t, c.floorPredicateCandidate(&upper))
	prior := c.integerSet
	used, stats := z.Budget().Used(), runtime.Stats()
	body = c.setIntegerWithInvocation(&out, &c.predicate.candidate, &c.predicate.setter)
	if !errors.Is(body, ErrAlgebraicPending) || c.predicate.setter.owner != nil {
		t.Fatal("refused wrapper SET issued receipt", body)
	}
	_ = c.finishPredicate(body)
	serviceMust(t, c.cleanupPredicate())
	if !reflect.DeepEqual(prior, c.integerSet) || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("predicate adopted or consumed older integer setter")
	}
	serviceMust(t, c.cleanupIntegerSet())
	if c.integerSet.active || c.integerSet.sourceCursor != 2 || c.integerSet.ownCursor != 2 {
		t.Fatal("older setter acknowledgement changed source cursor")
	}
	serviceMust(t, c.Del(&out))
}
