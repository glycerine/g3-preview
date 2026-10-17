package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestAlgebraicPredicateTraceOrderCacheAndFailbit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "closed"}[fail], func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
			var text strings.Builder
			refused := &linearTraceFailWriter{}
			var out io.Writer = &text
			if fail {
				out = refused
			}
			tags := diagnostic.NewTraceTagsWithWriter(out)
			tags.Enable("algebraic_bug")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			defer func() { serviceMust(t, c.config.Rationals.BindTrace(nil)) }()
			yes, e := c.IsRational(&root)
			serviceMust(t, e)
			if yes {
				t.Fatal("sqrt2 classified rational")
			}
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			if tags.ShouldTrace("algebraic_bug") {
				if r.flags>>3 != 2 {
					t.Fatal("TRACE failed to publish actual cached root index", r.flags)
				}
				if fail {
					if refused.calls != 1 {
						t.Fatal("sticky trace writer retried", refused.calls)
					}
				} else {
					bodies := []string{"is_rational(a):\n(#^2 - 2, 2)\n(1, 2)\n", "abs(an): 1, k: 1\n", "interval after refinement: (5/2^2, 3/2)\n"}
					pos := 0
					for _, body := range bodies {
						i := strings.Index(text.String()[pos:], body)
						if i < 0 {
							t.Fatal("missing or out-of-order source TRACE body", body, text.String())
						}
						pos += i + len(body)
					}
				}
			} else if text.Len() != 0 || refused.calls != 0 || r.flags>>3 != 0 {
				t.Fatal("erased TRACE formatted or cached root index")
			}
			if c.predicateDiagnosticRunning || c.predicateTrace.active {
				t.Fatal("TRACE left execution/owning scope active")
			}
			serviceMust(t, c.Del(&root))
		})
	}
}
func TestAlgebraicPredicateTraceCallbackPreservesOwner(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	output := &linearOperationProbeWriter{probe: func() {
		used := c.config.Rationals.Budget().Used()
		for _, e := range []error{c.Close(), c.CleanupRetained(), c.Del(&root), c.CleanupValue(&root)} {
			if !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("TRACE callback obtained owner", e)
			}
		}
		if used != c.config.Rationals.Budget().Used() || !c.predicateDiagnosticRunning || c.closed {
			t.Fatal("TRACE callback mutated owner")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	yes, e := c.IsRational(&root)
	serviceMust(t, e)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if yes || output.called != tags.ShouldTrace("algebraic_bug") {
		t.Fatal("TRACE callback/erasure", yes, output.called)
	}
	serviceMust(t, c.Del(&root))
}

// A deliberately oversized actual numeric size header permits real character
// scope construction, then refuses a decimal limb read beyond the object.
// This is a qualified bounds guard, not valid malloc exhaustion.
func TestAlgebraicPredicateTraceComputationAfterFailedStreamOwnsReceipt(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
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
	cell := z.Cell(c.isRationalTmp.Cell)
	size, e := cell.Backing.ReadWord(z.Allocator, 0)
	serviceMust(t, e)
	for i := size; i < cell.Capacity; i++ {
		serviceMust(t, z.RuntimeOwner().WriteUint32(cell.Backing.Raw, cell.Backing.Offset+8+4*uint64(i), 0))
	}
	serviceMust(t, z.RuntimeOwner().WriteUint32(cell.Backing.Raw, cell.Backing.Offset, cell.Capacity+1))
	refused := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	before, used := c.predicateTrace, z.Budget().Used()
	body := c.traceRationalPredicate(&root, 1, 65)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, z.RuntimeOwner().WriteUint32(cell.Backing.Raw, cell.Backing.Offset, size))
	if tags.ShouldTrace("algebraic_bug") {
		if !errors.Is(body, arena.ErrBounds) || refused.calls != 1 || !c.predicateTrace.insertions[0].Active() || !c.predicateTracePending(&c.predicate.trace) {
			t.Fatal("failed stream skipped later computation or lost its actual receipt", body, refused.calls)
		}
	} else {
		serviceMust(t, body)
		if refused.calls != 0 || !reflect.DeepEqual(before, c.predicateTrace) || used != z.Budget().Used() {
			t.Fatal("erased TRACE evaluated numeric operand")
		}
	}
	_ = c.finishPredicate(body)
	serviceMust(t, c.CleanupRetained())
	if c.predicateTrace.active || c.predicate.active {
		t.Fatal("TRACE owner remained active after acknowledgement")
	}
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPredicateTraceDisabledTagSkipsAllOperands(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	defer func() { serviceMust(t, c.config.Rationals.BindTrace(nil)) }()
	before, display := c.predicateTrace, c.rootDisplay
	used := c.config.Rationals.Budget().Used()
	c.config.Rationals.Budget().FailAfter(0)
	// No numeral/predicate exists. Disabled TRACE must not inspect operands or
	// require the saved-interval/coefficient declarations of the later phases.
	for phase := uint8(0); phase < 3; phase++ {
		serviceMust(t, c.traceRationalPredicate(nil, phase, 0))
	}
	c.config.Rationals.Budget().FailAfter(-1)
	if text.Len() != 0 || !reflect.DeepEqual(before, c.predicateTrace) || !reflect.DeepEqual(display, c.rootDisplay) || used != c.config.Rationals.Budget().Used() {
		t.Fatal("disabled tag acquired display/formatting authority or allocation")
	}
}

func TestAlgebraicPredicateInitialTracePrecedesSavedConstructor(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	control.allow = false
	_, e := c.IsRational(&root)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal("source root-index/refinement cancellation absent", e)
	}
	if tags.ShouldTrace("algebraic_bug") {
		if c.refinement.invocation != 0 || c.predicate.savedActive || c.predicate.active || !strings.Contains(text.String(), "is_rational(a):\n(#^2 - 2") {
			t.Fatal("initial trace failure reached older saved/refinement declarations", text.String())
		}
	} else if c.refinement.invocation == 0 {
		t.Fatal("erased trace did not reach ordinary refinement")
	}
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	if r.flags>>3 != 0 {
		t.Fatal("failed root-index computation published cache")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&root))
}
