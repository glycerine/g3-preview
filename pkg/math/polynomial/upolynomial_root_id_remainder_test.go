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

// Actual cancellation after a_m was physically constructed, followed by a
// separately qualified malformed requested-size guard in its source destructor.
func TestUPolynomialRootIDRemainderScalarStopProtectsVectors(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := p.config.Integer
	reached := false
	var ref unumeral.NumericCellRef
	var requested uint64
	p.config.Checkpoint = func() bool {
		q := &p.rootID
		if reached || !q.executing || !q.scalar.constructed {
			return true
		}
		ref = q.scalar.value.Cell
		if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
			return true
		}
		reached = true
		cell := z.Cell(ref)
		requested = cell.Backing.Requested
		cell.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(ref, cell)
		return false
	}
	p.config.Cooperation = func() error { return nil }
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	q := &p.rootID
	s := &q.scalar
	if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || s.sourceStop == nil || s.sourceCursor != 0 || !s.constructed || q.childStop == nil || !q.active || q.sourceCursor != ([3]uint32{}) || !q.live[0] || !q.live[1] || !q.live[2] || z.Cell(ref).Released {
		t.Fatal("scalar source stop lost older vectors", body, q.sourceCursor)
	}
	for _, state := range []*upolyVectorState{&q.remainder, &q.sequence.coefficients} {
		for _, value := range p.values(state)[:state.size] {
			if value.Cell != 0 && z.Cell(value.Cell).Released {
				t.Fatal("older vector numeral destroyed after scalar stop")
			}
		}
	}
	before, used, stats := *s, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := q.finishRemainderScalar(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("source scalar finalizer replayed")
	}
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if q.active || s.constructed || s.sourceCursor != 0 || s.ownCursor != 1 || s.sourceStop == nil || q.sourceCursor != ([3]uint32{}) {
		t.Fatal("own scalar cleanup advanced source cursors")
	}
	id, e := p.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if id != 0 {
		t.Fatal("independent root count below lower1", id)
	}
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialRootIDRemainderCopyOwnsActualAllocation(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	var receipt upolyRootIDInvocationReceipt
	q, e := p.beginRootID(v, &receipt)
	serviceMust(t, e)
	// Execute the real retained prefix before remainder's initial SET.
	serviceMust(t, q.sturmPrefix(v))
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := q.sturmCore()
	body = q.finishVector(1, body)
	body = q.finish(body)
	z := p.config.Integer
	if body == nil || !q.active || !z.MatchesCopyAllocationInvocation(&q.remainderCopy) || q.scalar.constructed || q.scalar.invocation != 0 {
		t.Fatal("remainder SET refusal lost receipt or constructed a_m early", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	source := q.sourceCursor
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if q.active || source != q.sourceCursor || z.MatchesCopyAllocationInvocation(&q.remainderCopy) {
		t.Fatal("remainder SET acknowledgement changed source history")
	}
	serviceMust(t, c.Del(&root))
}

// Real source cancellation and real runtime deallocation accounting refusal;
// no malformed numeral or synthetic pending/ownership flag is installed.
func TestUPolynomialRootIDRemainderScalarActualReleaseGuard(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := p.config.Integer
	reached := false
	var ref unumeral.NumericCellRef
	var lane umemory.Block
	p.config.Checkpoint = func() bool {
		q := &p.rootID
		if reached || !q.executing || !q.scalar.constructed {
			return true
		}
		ref = q.scalar.value.Cell
		if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
			return true
		}
		reached = true
		contextTLSNearMinimum(t, runtime)
		var e error
		lane, e = runtime.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000)
		if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		return false
	}
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	q := &p.rootID
	s := &q.scalar
	if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || s.sourceStop == nil || s.sourceCursor != 0 || !s.constructed || !z.Cell(ref).Released || q.childStop == nil || q.sourceCursor != ([3]uint32{}) {
		t.Fatal("actual a_m release guard did not protect vectors", body, q.sourceCursor)
	}
	before, used, stats := *s, z.Budget().Used(), runtime.Stats()
	if e := q.finishRemainderScalar(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("consumed scalar release replayed")
	}
	contextTLSFinish(t, runtime, func() error { return p.cleanupRootIDWithReceipt(&receipt) })
	if q.active || s.constructed || s.sourceCursor != 0 || s.ownCursor != 1 || q.sourceCursor != ([3]uint32{}) {
		t.Fatal("own scalar cleanup changed source cursors")
	}
	contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
}

func TestUPolynomialRootIDRemainderSubMulFailureRetainsUnjoinedTemporary(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || p == nil || !p.rootID.executing || !p.rootID.scalar.constructed {
					return f
				}
				z := p.config.Integer
				if !z.MatchesAddMulInvocation(&p.rootID.scalar.addMul) {
					return f
				}
				ref := z.AddMulInvocation.Tmp.Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				context := p.rootID.context()
				probe := unumeral.NewInt(0)
				used := z.Budget().Used()
				if context.rootIDFrame(z) != nil || p.rootID.NumericRoots(z, nil) != nil {
					t.Fatal("failure transport acquired paused numerical frame")
				}
				if e := z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7); e == nil || probe.Val != 0 || used != z.Budget().Used() {
					t.Fatal("failure transport numerical context mutated owner", e)
				}
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
			c, p, runtime = algebraicNegTLSFixture(t, transport)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			var receipt upolyRootIDInvocationReceipt
			q, e := p.beginRootID(v, &receipt)
			serviceMust(t, e)
			serviceMust(t, q.sturmPrefix(v))
			z := p.config.Integer
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			body := q.sturmCore()
			body = q.finishVector(1, body)
			body = q.finish(body)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			s := &q.scalar
			if reached {
				if !errors.Is(body, arena.ErrBounds) || s.childStop != nil || s.sourceStop == nil || !s.constructed || s.sourceCursor != 0 || !z.MatchesAddMulInvocation(&s.addMul) || !z.Cell(s.value.Cell).Released || q.childStop == nil || q.sourceCursor != ([3]uint32{}) {
					t.Fatal("SubMul body failure/AM destructor stop lost source order", body, q.sourceCursor)
				}
				if z.AddMulInvocation.Stage == 3 || z.AddMulInvocation.SourceCause != nil || !unumeral.ValidNumericRef(z.Cells, z.AddMulInvocation.Tmp.Cell) || z.Cell(z.AddMulInvocation.Tmp.Cell).Released {
					t.Fatal("plain RAW temporary was destroyed without reaching source END")
				}
				before, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if e := q.finishRemainderScalar(nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("scalar finalizer replayed after SubMul stop")
				}
			}
			contextTLSFinish(t, runtime, func() error { return p.cleanupRootIDWithReceipt(&receipt) })
			if reached && (q.active || s.constructed || s.sourceCursor != 0 || s.ownCursor != 1 || q.sourceCursor != ([3]uint32{})) {
				t.Fatal("SubMul owning cleanup advanced scalar/vector source cursors")
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
	}
	if !reached {
		t.Fatal("actual SubMul body failure and scalar release guard not reached")
	}
}

func TestUPolynomialRootIDRemainderScalarCopyExactAcknowledgement(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	var pad umemory.Block
	armed := false
	p.config.Checkpoint = func() bool {
		if !armed && p.rootID.scalar.constructed && p.rootID.scalar.value.Cell == 0 {
			armed = true
			pad = algebraicCellSourceTLSBoundary(t, c)
		}
		return true
	}
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	q := &p.rootID
	s := &q.scalar
	z := p.config.Integer
	if !armed || body == nil || !q.active || !z.MatchesCopyAllocationInvocation(&s.copy) || s.invocation != 1 || s.constructed || s.sourceCursor != 1 || s.sourceStop != nil || q.childStop != nil {
		t.Fatal("actual a_m SET allocation/constructor/destructor order", body, s.sourceCursor)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	cursors := q.sourceCursor
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if q.active || s.sourceCursor != 1 || q.sourceCursor != cursors || z.MatchesCopyAllocationInvocation(&s.copy) {
		t.Fatal("a_m copy ACK changed source history")
	}
	id, e := p.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if id != 0 {
		t.Fatal("independent root count after first-copy refusal", id)
	}
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialRootIDRemainderSubMulEndStopProtectsScalar(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := p.config.Integer
	// Real capacity growth and multiplication warm the actual destinations and
	// shared scratch. Neither operation changes the polynomial's numeric values.
	for i := uint32(0); i < v.count; i++ {
		a, e := p.read(v, i)
		serviceMust(t, e)
		serviceMust(t, z.EnsureOwnedCapacity(a, 16))
	}
	var warm, product unumeral.Int
	digits := make([]uint32, 17)
	digits[16] = 1
	serviceMust(t, z.SetLimbs(&warm, digits))
	serviceMust(t, z.Mul(&warm, &warm, &product))
	serviceMust(t, z.Del(&product))
	serviceMust(t, z.Del(&warm))
	reached := false
	var lane umemory.Block
	p.config.Checkpoint = func() bool {
		if reached {
			return true
		}
		reached = true
		contextTLSNearMinimum(t, runtime)
		var e error
		lane, e = runtime.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000)
		if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		return true
	}
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	q := &p.rootID
	s := &q.scalar
	if !reached || !errors.Is(body, arena.ErrBounds) || s.childStop == nil || s.sourceStop != nil || !s.constructed || s.sourceCursor != 0 || !z.MatchesAddMulInvocation(&s.addMul) || z.AddMulInvocation.Stage != 3 || z.AddMulInvocation.SourceCause == nil || z.Cell(s.value.Cell).Released || q.sourceCursor != ([3]uint32{}) {
		t.Fatal("actual SubMul END stop did not protect a_m", body, z.AddMulInvocation.Stage, s.sourceCursor)
	}
	if !z.Cell(z.AddMulInvocation.Tmp.Cell).Released {
		t.Fatal("source RAW END did not consume physical temporary")
	}
	contextTLSFinish(t, runtime, func() error { return p.cleanupRootIDWithReceipt(&receipt) })
	if q.active || s.constructed || s.sourceCursor != 0 || s.ownCursor != 1 || q.sourceCursor != ([3]uint32{}) {
		t.Fatal("SubMul END own acknowledgement changed source cursors")
	}
	contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
}

func TestUPolynomialRootIDNumericalContextRejectsCheckpointAndRetirement(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := p.config.Integer
	probed := false
	var context integerAllocationInvocationContext
	output := unumeral.NewInt(0)
	p.config.Checkpoint = func() bool {
		if probed {
			return true
		}
		probed = true
		context = p.rootID.context()
		var untouched upolyVectorState
		if e := p.setViewWithSourceContext(&untouched, v, context); !errors.Is(e, ErrUPolynomialOwner) || untouched != (upolyVectorState{}) {
			t.Fatal("checkpoint acquired a native vector before context validation", e)
		}
		if e := p.derivativeStateWithSourceContext(v, &untouched, context); !errors.Is(e, ErrUPolynomialOwner) || untouched != (upolyVectorState{}) {
			t.Fatal("checkpoint acquired derivative vector before context validation", e)
		}
		if e := p.normalizeWithSourceContext(v, context); !errors.Is(e, ErrUPolynomialOwner) {
			t.Fatal("checkpoint acquired normalization context", e)
		}
		used := z.Budget().Used()
		if context.rootIDFrame(z) != nil || p.rootID.NumericRoots(z, nil) != nil {
			t.Fatal("checkpoint acquired paused numerical frame")
		}
		e := z.WithAllocationContext(context.utility(z)).SetUint32(&output, 7)
		if e == nil || output.Val != 0 || output.Cell != 0 || used != z.Budget().Used() {
			t.Fatal("checkpoint numerical context mutated owner", e)
		}
		return true
	}
	p.config.Cooperation = func() error { return nil }
	id, e := p.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if !probed || id != 0 {
		t.Fatal("real numerical body did not resume", id)
	}
	if context.rootIDFrame(z) != nil {
		t.Fatal("retired invocation retained context authority")
	}
	if e := z.WithAllocationContext(context.utility(z)).SetUint32(&output, 9); e == nil || output.Val != 0 {
		t.Fatal("retired context accepted numerical write", e)
	}
	var untouched upolyVectorState
	used := z.Budget().Used()
	if e := p.derivativeStateWithSourceContext(v, &untouched, context); !errors.Is(e, ErrUPolynomialOwner) || untouched != (upolyVectorState{}) || used != z.Budget().Used() {
		t.Fatal("retired derivative context changed storage", e)
	}
	serviceMust(t, c.Del(&root))
}
