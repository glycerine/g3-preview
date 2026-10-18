package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestSimpleAlgebraicValuesOwnedCopiesDuplicatesAndEvaluation(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	values, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, values.Close()) }()
	m, e := values.Manager()
	serviceMust(t, e)
	if m != c {
		t.Fatal("mapping manager identity")
	}
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	serviceMust(t, values.Push(0, &root))
	serviceMust(t, c.Del(&root))
	var three Algebraic
	n := unumeral.NewInt(3)
	serviceMust(t, c.SetInteger(&three, &n))
	serviceMust(t, values.Push(0, &three))
	serviceMust(t, c.Del(&three))
	first, e := values.Algebraic(0)
	serviceMust(t, e)
	algebraicExpectBetween(t, c, first, 1, 2)
	yes, e := values.Contains(0)
	serviceMust(t, e)
	no, e := values.Contains(1)
	serviceMust(t, e)
	if !yes || no {
		t.Fatal("contains", yes, no)
	}
	// Evaluate x^2-2 using the real mapping and recursive algebraic evaluator.
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}})
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	serviceMust(t, pm.EvaluateAlgebraic(p, c, values, &out))
	zero, e := c.IsZero(&out)
	serviceMust(t, e)
	if !zero {
		t.Fatal("mapping evaluation of sqrt(2)^2-2")
	}
	if e = c.Close(); !errors.Is(e, ErrAlgebraicLive) {
		t.Fatal("mapping did not retain numerical owner", e)
	}
}

func TestSimpleAlgebraicValuesMissingAndFailedPushPrefix(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	values, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "C")
	missing, e := values.Algebraic(99)
	if polynomialSourceDebug {
		serviceMust(t, e)
		if missing == nil || *missing != (Algebraic{}) {
			t.Fatal("source static zero", missing)
		}
	} else if !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
		t.Fatal("missing lookup diagnostic", e)
	}
	var a Algebraic
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&a, &one))
	c.config.Rationals.Budget().FailAfter(0)
	e = values.Push(7, &a)
	c.config.Rationals.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("expected actual value-copy refusal", e)
	}
	contains, e := values.Contains(7)
	serviceMust(t, e)
	if !contains || len(values.variables) != 1 || values.values.Len() != 1 {
		t.Fatal("source variable/value prefix changed")
	}
	retained, e := values.Algebraic(7)
	serviceMust(t, e)
	if retained == nil || *retained != (Algebraic{}) {
		t.Fatal("failed copy lost its published zero slot", retained)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, values.CleanupRetained())
	// The retained zero slot remains a real mapping entry, and later insertion
	// keeps variable/value indices aligned.
	serviceMust(t, values.Push(8, &a))
	copied, e := values.Algebraic(8)
	serviceMust(t, e)
	equal, e := c.Equal(copied, &a)
	serviceMust(t, e)
	if !equal {
		t.Fatal("continued mapping lost alignment")
	}
	serviceMust(t, values.Close())
	serviceMust(t, c.Del(&a))
	// A fresh mapping remains independently reusable.
	fresh, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	serviceMust(t, fresh.Push(1, &a))
	serviceMust(t, fresh.Close())
}

func TestSimpleAlgebraicValuesCopyReservationPrefixes(t *testing.T) {
	success, retained, stopped, ownedReceipt := false, false, false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, up, _ := algebraicCompareFixture(t)
			// A genuine BIG-coefficient defining polynomial forces actual
			// coefficient allocations after the copied root cell is published.
			coefficients := []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(0)}
			z := c.config.Rationals.Integers()
			serviceMust(t, z.SetString(&coefficients[0], "-20000000000000000000000000000000000000001"))
			serviceMust(t, z.SetString(&coefficients[2], "10000000000000000000000000000000000000000"))
			poly, e := up.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, up.SetInts(poly, coefficients))
			view, e := up.wholeView(poly)
			serviceMust(t, e)
			lower, upper := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
			var root Algebraic
			serviceMust(t, c.SetRoot(&root, view, &lower, &upper, true))
			values, e := NewSimpleAlgebraicValues(c)
			serviceMust(t, e)
			budget := c.config.Rationals.Budget()
			budget.FailAfter(ordinal)
			var body error
			last := uint32(0)
			for last = 0; last < 32; last++ {
				body = values.Push(last, &root)
				if body != nil {
					break
				}
			}
			budget.FailAfter(-1)
			expected := int(last) + 1
			if body == nil {
				expected = 32
				last = 31
			}
			if len(values.variables) != expected || int(values.values.Len()) != expected {
				t.Fatal("copy failure changed published slot count", ordinal, body)
			}
			cell, e := values.Algebraic(last)
			serviceMust(t, e)
			if body == nil {
				success = true
				eq, e := c.Equal(cell, &root)
				serviceMust(t, e)
				if !eq {
					t.Fatal("successful copy value")
				}
			} else {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if cell.slot != 0 {
					retained = true
					l, e := c.location(cell)
					serviceMust(t, e)
					r, e := c.load(l)
					serviceMust(t, e)
					if l.kind == 1 && l.coefficientInitialized < r.size {
						if e = values.Close(); !errors.Is(e, arena.ErrBounds) {
							t.Fatal("source destructor did not stop at unconstructed coefficient", e)
						}
						stopped = true
					}
				}
			}
			// The core acknowledges only its exact issued cell and numeric
			// coefficient allocations; no global allocator cleanup is used.
			if z.MatchesCopyAllocationInvocation(&c.coefficientCopy) {
				ownedReceipt = true
				var untouched Algebraic
				if e := c.Copy(&untouched, &root); !errors.Is(e, ErrAlgebraicPending) || untouched != (Algebraic{}) {
					t.Fatal("pending copy replaced its own receipt", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, values.CleanupRetained())
			serviceMust(t, values.Close())
			// Retry in a fresh owning mapping after acknowledging failed numerical work.
			fresh, e := NewSimpleAlgebraicValues(c)
			serviceMust(t, e)
			serviceMust(t, fresh.Push(12, &root))
			cell, e = fresh.Algebraic(12)
			serviceMust(t, e)
			eq, e := c.Equal(cell, &root)
			serviceMust(t, e)
			if !eq {
				t.Fatal("retry value")
			}
			serviceMust(t, fresh.Close())
			serviceMust(t, c.Del(&root))
			serviceMust(t, up.DelPolynomial(&poly))
			for i := range coefficients {
				serviceMust(t, z.Del(&coefficients[i]))
			}
		})
		if t.Failed() {
			return
		}
	}
	if !success || !retained || !stopped || !ownedReceipt {
		t.Fatal("missing success/published partial-root/stopped-destructor/issued-receipt frontier", success, retained, stopped, ownedReceipt)
	}
}

func TestSimpleAlgebraicValuesConstructorRollbackAndCopiedOwner(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	budget := c.config.Rationals.Budget()
	for prefix := int64(0); prefix < 3; prefix++ {
		before, vectors := budget.Used(), c.vectors
		budget.FailAfter(prefix)
		values, body := NewSimpleAlgebraicValues(c)
		budget.FailAfter(-1)
		if prefix < 2 {
			if !numericPhysicalReservationRefusal(body) || values != nil || budget.Used() != before || c.vectors != vectors {
				t.Fatal("constructor refusal changed borrowed owner", prefix, body)
			}
			continue
		}
		serviceMust(t, body)
		if c.vectors != vectors+1 {
			t.Fatal("completed mapping did not own its vector")
		}
		copied := *values
		attempts := []func() error{
			func() error { _, e := copied.Manager(); return e },
			func() error { _, e := copied.Contains(0); return e },
			func() error { _, e := copied.Algebraic(0); return e },
			func() error { return copied.Push(0, nil) }, copied.Close, copied.CleanupRetained,
		}
		used := budget.Used()
		for _, attempt := range attempts {
			if e := attempt(); !errors.Is(e, ErrAlgebraicOwner) {
				t.Fatal("copied mapping acquired ownership", e)
			}
		}
		if budget.Used() != used || values.closed || values.values.Len() != 0 || c.vectors != vectors+1 {
			t.Fatal("copied owner changed original")
		}
		serviceMust(t, values.Close())
		serviceMust(t, values.Close())
		if budget.Used() != before || c.vectors != vectors {
			t.Fatal("mapping close did not refund exact constructor ownership")
		}
		if _, e := values.Manager(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("closed mapping borrowed manager", e)
		}
	}
}
