package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

// Actual returning failure transport and signed-host destructor guard. The
// saved interval is constructed before the reached refinement/Horner child.
func TestAlgebraicRationalPredicateRefinementStopRetainsSavedInterval(t *testing.T) {
	algebraicPredicateRefinementStopCase(t, true, false)
}
func TestAlgebraicRationalPredicateDoesNotAdoptOlderRefinementStop(t *testing.T) {
	algebraicPredicateRefinementStopCase(t, false, false)
}
func TestAlgebraicToOwnedRationalPredicateRefinementStop(t *testing.T) {
	algebraicPredicateRefinementStopCase(t, true, true)
}
func algebraicPredicateRefinementStopCase(t *testing.T, predicateFirst, ownedWrapper bool) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || c == nil || !c.refinement.executing || (predicateFirst && !c.predicate.active) {
					return f
				}
				z := c.config.Rationals.Integers()
				if z.AddMulInvocation.Tmp.Cell == 0 || z.IsFailureRef(z.AddMulInvocation.Tmp.Cell) {
					return f
				}
				armed = false
				reached = true
				serviceMust(t, runtime.SetMaxSize(0))
				contextTLSNearMinimum(t, runtime)
				var e error
				lane, e = runtime.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				return f
			}
			var p *UPolynomialManager
			c, p, runtime = algebraicNegTLSFixture(t, transport)
			root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
			var global *unumeral.RationalService
			var out unumeral.OwnedRational
			if ownedWrapper {
				global, _ = serviceNew(t)
				var e error
				out, e = global.NewValue()
				serviceMust(t, e)
				serviceMust(t, global.SetInt64(&out, 99))
			}
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent", runtime.State())
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			var body error
			if ownedWrapper {
				body = c.ToOwnedRational(&root, global, &out)
			} else if predicateFirst {
				_, body = c.IsRational(&root)
			} else {
				body = c.RefineToPrecision(&root, 1)
			}
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			if reached && predicateFirst {
				s := &c.predicate
				if !errors.Is(body, arena.ErrBounds) || s.childStop == nil || s.cause == nil || !s.active || s.stage != 0 || s.restored || !s.savedConstructed || !c.matchesRefinement(&s.refinement) {
					t.Fatal("older predicate finalizer crossed owned refinement stop", body, s.stage, s.active, s.restored)
				}
				z := c.config.Rationals.Integers()
				for _, v := range [2]*unumeral.Int{s.saved.BorrowLower().BorrowNumerator(), s.saved.BorrowUpper().BorrowNumerator()} {
					if !unumeral.ValidNumericRef(z.Cells, v.Cell) || z.Cell(v.Cell).Released {
						t.Fatal("actual saved interval released before owning acknowledgement")
					}
				}
				before, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if _, e := c.IsRational(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("stopped predicate replaced", e)
				}
				if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("replacement mutated retained predicate")
				}
				if e := c.cleanupPredicate(); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("older owner cleaned before younger refinement", e)
				}
				if !p.binaryEvaluation.active {
					t.Fatal("actual reached Horner child was not retained")
				}
				if ownedWrapper && (!c.toOwnedRational.active || c.toOwnedRational.constructed || out.Raw.Num != unumeral.NewInt(99)) {
					t.Fatal("wrapper published output or lost actual predicate stop")
				}
			}
			if reached && !predicateFirst {
				old := c.refinement
				oldEval := p.binaryEvaluation
				_, e := c.IsRational(&root)
				if e == nil || c.predicate.refinement.owner != nil || c.predicate.childStop != nil {
					t.Fatal("predicate adopted refused older refinement", e)
				}
				if !reflect.DeepEqual(old, c.refinement) || !reflect.DeepEqual(oldEval, p.binaryEvaluation) {
					t.Fatal("refused predicate changed older actual refinement")
				}
			}
			contextTLSFinish(t, runtime, c.CleanupRetained)
			if reached && predicateFirst && (c.predicate.active || c.predicate.stage != 0 || c.predicate.ownStage != 5 || c.predicate.childStop == nil) {
				t.Fatal("owning predicate continuation lost source stop")
			}
			if c.predicate.active {
				t.Fatal("owning cleanup retained predicate")
			}
			if ownedWrapper {
				if c.toOwnedRational.active || out.Raw.Num != unumeral.NewInt(99) {
					t.Fatal("core cleanup retained wrapper or consumed caller output")
				}
				serviceMust(t, global.Del(&out))
			}
			contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
			for _, block := range []*umemory.Block{&lane, &pad} {
				contextTLSFinish(t, runtime, func() error {
					if *block == (umemory.Block{}) {
						return nil
					}
					return runtime.Deallocate(block)
				})
			}
		})
		if t.Failed() {
			return
		}
	}
	if !reached {
		t.Fatal("no actual nested refinement destructor stop reached")
	}
}

func TestAlgebraicRationalPredicateFailedSavedConstructorHasNoSourceDestructor(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := c.IsRational(&root)
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	s := &c.predicate
	if body == nil || !s.active || !s.savedActive || s.savedConstructed || s.restored || s.stage != 5 || s.refinement.owner != nil {
		t.Fatal("failed saved constructor ran a source destructor", body, s.stage)
	}
	if !c.config.Rationals.Integers().MatchesCopyAllocationInvocation(&s.savedCopy) {
		t.Fatal("failed constructor did not retain its actual copy allocation receipt")
	}
	if _, e := c.IsRational(&root); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("partial saved constructor replaced", e)
	}
	serviceMust(t, c.CleanupRetained())
	if s.active || s.ownStage != 5 || s.stage != 5 || s.savedConstructed {
		t.Fatal("partial constructor owning cleanup lost source cursor")
	}
	p.config.Checkpoint = func() bool { return true }
	p.config.Cooperation = func() error { return nil }
	yes, e := c.IsRational(&root)
	serviceMust(t, e)
	if yes {
		t.Fatal("sqrt(2) became rational after constructor acknowledgement")
	}
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicRationalPredicateSavedUpperCopyFailureRetainsLowerForOwnCleanup(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 256 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
			z := c.config.Rationals.Integers()
			runtime := z.RuntimeOwner()
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent", runtime.State())
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			_, body := c.IsRational(&root)
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			s := &c.predicate
			ref := s.saved.BorrowLower().BorrowNumerator().Cell
			if body != nil && s.active && !s.savedConstructed && unumeral.ValidNumericRef(z.Cells, ref) {
				reached = true
				if s.stage != 5 || !z.MatchesCopyAllocationInvocation(&s.savedCopy) || z.Cell(ref).Released {
					t.Fatal("partial saved constructor lost lower/upper ownership")
				}
			}
			serviceMust(t, c.CleanupRetained())
			if reached && (s.active || unumeral.ValidNumericRef(z.Cells, ref)) {
				t.Fatal("owning cleanup did not release lower of failed saved constructor")
			}
			serviceMust(t, c.Del(&root))
		})
		if t.Failed() {
			return
		}
	}
	if !reached {
		t.Fatal("actual upper-copy refusal after completed lower copy not reached")
	}
}

func TestAlgebraicRationalPredicateOwnsLeadingCoefficientCopy(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 256 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			z := c.config.Rationals.Integers()
			runtime := z.RuntimeOwner()
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent", runtime.State())
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			_, body := c.IsRational(&root)
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			s := &c.predicate
			if body != nil && z.MatchesCopyAllocationInvocation(&s.leadingCopy) {
				reached = true
				if !s.active || !s.savedConstructed || s.refinement.owner != nil {
					t.Fatal("leading-copy debt lost before refinement")
				}
				if _, e := c.IsRational(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("leading-copy owner replaced", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if reached && (s.active || z.MatchesCopyAllocationInvocation(&s.leadingCopy)) {
				t.Fatal("predicate cleanup did not acknowledge actual leading copy")
			}
			// The cached destination is deliberately not repaired after destructive
			// setter failure. Only its owning core later destroys that cache.
			serviceMust(t, c.Del(&root))
		})
		if t.Failed() {
			return
		}
	}
	if !reached {
		t.Fatal("actual leading-coefficient copy refusal not reached")
	}
}
