package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicPredicateScaledBoundsAllocationPrefixes(t *testing.T) {
	success, issued, refused := false, false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			serviceMust(t, c.beginPredicate())
			serviceMust(t, c.constructPredicateSaved(&root, &r))
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			lead, e := c.config.Polynomials.read(v, r.size-1)
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			serviceMust(t, z.CopyFromOwnerWithReceipt(&c.isRationalTmp, z, lead, &c.predicate.leadingCopy))
			serviceMust(t, z.Abs(&c.isRationalTmp))
			z.Budget().FailAfter(ordinal)
			body := c.scalePredicateBounds(&r)
			z.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			if body != nil {
				refused = true
			}
			s := &c.predicate
			if unumeral.NumericAllocationPending(&s.numericRoots) {
				issued = true
				if e := c.scalePredicateBounds(&r); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("replaced outstanding arithmetic allocation", e)
				}
			}
			if body == nil {
				success = true
				for i, value := range []*unumeral.BinaryRational{&s.lower, &s.upper} {
					text, e := z.String(value.BorrowNumerator())
					serviceMust(t, e)
					want := []string{"18446744073709551616", "18446744073709551618"}[i]
					if text != want || value.K() != 0 {
						t.Fatal("independent scaled bound", i, text, value.K())
					}
				}
			}
			// Source finalization may release constructed values but must retain any
			// unmatched numerical allocation until the explicit owning continuation.
			_ = c.finishPredicate(body)
			serviceMust(t, c.CleanupRetained())
			if unumeral.NumericAllocationPending(&c.predicate.numericRoots) {
				t.Fatal("arithmetic allocation retained")
			}
			serviceMust(t, c.Del(&root))
		})
	}
	if !success {
		t.Fatal("did not reach complete scaled bounds")
	}
	if !refused {
		t.Fatal("did not reach a physical reservation refusal")
	}
	t.Log("unjoined numerical allocation observed", issued)
}

func TestAlgebraicPredicateScaledBoundsRejectCallbackReentry(t *testing.T) {
	var c *AlgebraicRootCore
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.predicate.arithmeticExecuting {
			return f
		}
		reached = true
		if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("callback cleanup", e)
		}
		if e := c.Close(); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("callback close", e)
		}
		if e := c.cleanupPredicate(); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("callback predicate cleanup", e)
		}
		if e := c.finishPredicate(nil); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("callback predicate destruction", e)
		}
		return f
	}
	c, _, runtime := algebraicNegTLSFixture(t, transport)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.constructPredicateSaved(&root, &r))
	z := c.config.Rationals.Integers()
	serviceMust(t, z.CopyFromOwnerWithReceipt(&c.isRationalTmp, z, &z.Two64, &c.predicate.leadingCopy))
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.scalePredicateBounds(&r)
	if !reached || body == nil || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual scaled arithmetic failure callback absent", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	_ = c.finishPredicate(body)
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&root))
}

// Construct -(2^256+1)/2 without warming arithmetic scratch through parsing.
// The source correction changes the truncated -2^255 into -2^255-1.
func TestAlgebraicPredicateNegativeFloorAllocationPrefixes(t *testing.T) {
	success, refused, correction := false, false, false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			z, b := c.config.Rationals.Integers(), c.config.Binary
			n := unumeral.NewInt(0)
			digits := make([]uint32, 9)
			digits[0], digits[8] = 1, 1
			serviceMust(t, z.SetLimbs(&n, digits))
			serviceMust(t, z.Neg(&n))
			var input unumeral.BinaryRational
			serviceMust(t, b.SetIntegerPower(&input, &n, 1))
			serviceMust(t, z.Del(&n))
			serviceMust(t, c.beginPredicate())
			z.Budget().FailAfter(ordinal)
			body := c.floorPredicateCandidate(&input)
			z.Budget().FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				text, e := z.String(&c.predicate.candidate)
				if e == nil && text == "-57896044618658097711785492504343953926634992332820282019728792003956564819968" {
					correction = true
				}
			} else {
				success = true
				text, e := z.String(&c.predicate.candidate)
				serviceMust(t, e)
				if text != "-57896044618658097711785492504343953926634992332820282019728792003956564819969" {
					t.Fatal("negative floor", text)
				}
			}
			_ = c.finishPredicate(body)
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, b.Del(&input))
		})
	}
	if !success || !refused || (!polynomialSourceDebug && !correction) {
		t.Fatal("missing floor success/copy or correction refusal", success, refused, correction)
	}
}
