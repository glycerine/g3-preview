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

// A real canceled Sturm remainder body plus an explicitly malformed sequence
// coefficient's requested-size guard proves source and own destruction order.
func TestUPolynomialRootIDSequenceDestructorRetainsCursorAndBody(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	reached := false
	var first, bad unumeral.NumericCellRef
	var requested uint64
	p.config.Checkpoint = func() bool {
		q := &p.rootID
		if reached || !q.executing || !q.live[0] || len(q.sequence.sizes) < 2 {
			return true
		}
		a := p.values(&q.sequence.coefficients)
		first, bad = a[0].Cell, a[1].Cell
		if !unumeral.ValidNumericRef(z.Cells, first) || !unumeral.ValidNumericRef(z.Cells, bad) {
			t.Fatal("real copied sequence coefficients absent")
		}
		reached = true
		for _, e := range []error{p.Close(), p.SourceFinalize(), p.CleanupRootID()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("root-ID checkpoint obtained lifecycle", e)
			}
		}
		cell := z.Cell(bad)
		requested = cell.Backing.Requested
		cell.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(bad, cell)
		return false
	}
	p.config.Cooperation = func() error { return nil }
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	q := &p.rootID
	if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || !p.matchesRootID(&receipt) || !q.active || q.sourceCursor[2] != 1 || q.sourceStop == nil || q.live[0] || q.live[1] || !q.live[2] || unumeral.ValidNumericRef(z.Cells, first) || z.Cell(bad).Released {
		t.Fatal("sequence source prefix/body error lost", body, q.sourceCursor)
	}
	original, e := p.read(v, 0)
	serviceMust(t, e)
	if !unumeral.ValidNumericRef(z.Cells, original.Cell) || z.Cell(original.Cell).Released {
		t.Fatal("root-ID destruction consumed borrowed polynomial")
	}
	state, used, stats := *q, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := q.finishVector(2, nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(state, *q) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("source sequence destructor replayed")
	}
	if _, e := p.GetRootID(v, r.interval.BorrowLower()); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("stopped root-ID replaced", e)
	}
	var refused upolyRootIDInvocationReceipt
	_, e = p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &refused)
	if !errors.Is(e, ErrUPolynomialCleanupPending) || p.matchesRootID(&refused) {
		t.Fatal("refused root-ID entry issued receipt", e)
	}
	serviceMust(t, p.cleanupRootIDWithReceipt(&refused))
	if !reflect.DeepEqual(state, *q) {
		t.Fatal("refused receipt acquired retained sequence")
	}
	count := q.sequence.coefficients.size
	cell := z.Cell(bad)
	cell.Backing.Requested = requested
	z.Store(bad, cell)
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if q.active || q.sourceCursor[2] != 1 || q.ownCursor[2] != count || q.sourceStop == nil {
		t.Fatal("own sequence cleanup moved source cursor")
	}
	id, e := p.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if id != 0 {
		t.Fatal("independent count of roots below lower1", id)
	}
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialRootIDInitialCopyExactOwnership(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	pad := algebraicCellSourceTLSBoundary(t, c)
	var receipt upolyRootIDInvocationReceipt
	_, body := p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	if body == nil || !p.matchesRootID(&receipt) || !p.rootID.active || !z.MatchesCopyAllocationInvocation(&p.rootID.initialCopy) {
		t.Fatal("actual sequence initial-copy allocation missing", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	cursors := p.rootID.sourceCursor
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if p.rootID.active || cursors != p.rootID.sourceCursor || z.MatchesCopyAllocationInvocation(&p.rootID.initialCopy) {
		t.Fatal("initial-copy ownership/cursor lost")
	}
	id, e := p.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if id != 0 {
		t.Fatal("retry root-ID", id)
	}
	serviceMust(t, c.Del(&root))
}

// Qualified returning OOM/signed-host accounting guard after the real Sturm
// prefix, at the nested binary-Horner phase of the same root-ID invocation.
func TestUPolynomialRootIDHornerStopProtectsOlderSequence(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || p == nil || !p.rootID.executing || !p.binaryEvaluation.executing {
					return f
				}
				z := p.config.Integer
				ref := p.binaryEvaluation.values[0].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				for _, e := range []error{p.Close(), p.SourceFinalize(), p.CleanupRootID()} {
					if !errors.Is(e, ErrUPolynomialCleanupPending) {
						t.Fatal("Horner callback obtained root-ID lifecycle", e)
					}
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
			serviceMust(t, q.sturm(v))
			z := p.config.Integer
			original := append([]unumeral.Int(nil), p.values(&q.sequence.coefficients)...)
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			_, body := q.evaluate(r.interval.BorrowLower())
			body = q.finish(body)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			if reached {
				if !errors.Is(body, arena.ErrBounds) || q.childStop == nil || !p.matchesBinaryEvaluationReceipt(&q.evaluation) || !p.binaryEvaluation.active || q.sourceCursor[2] != 0 || !q.live[2] || !q.active || !reflect.DeepEqual(original, p.values(&q.sequence.coefficients)) {
					t.Fatal("nested Horner stop lost older sequence", body, q.sourceCursor)
				}
				for _, value := range original {
					if value.Cell != 0 && z.Cell(value.Cell).Released {
						t.Fatal("sequence coefficient physically destroyed after younger stop")
					}
				}
				before, used, stats := *q, z.Budget().Used(), runtime.Stats()
				if e := q.finishVector(2, nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("source sequence finalization replayed")
				}
			}
			count := q.sequence.coefficients.size
			contextTLSFinish(t, runtime, func() error { return p.cleanupRootIDWithReceipt(&receipt) })
			if reached && (q.active || q.sourceCursor[2] != 0 || q.ownCursor[2] != count || q.childStop == nil || p.binaryEvaluation.active) {
				t.Fatal("own cleanup failed child-first order or moved source cursor")
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
		t.Fatal("nested root-ID Horner source stop not reached")
	}
}

type rootIDArmingSink struct {
	algebraicRootTextSink
	arm func()
}

func (s *rootIDArmingSink) AppendNumeralText(v string) error {
	if v == "36893488147419103234" && s.arm != nil {
		arm := s.arm
		s.arm = nil
		arm()
	}
	return s.algebraicRootTextSink.AppendNumeralText(v)
}
func TestAlgebraicRootDisplayOwnsAcceptedRootIDCopy(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	var pad umemory.Block
	armed := false
	sink := &rootIDArmingSink{arm: func() { armed = true; pad = algebraicCellSourceTLSBoundary(t, c) }}
	body := c.DisplayRootInto(sink, &root)
	z := c.config.Rationals.Integers()
	if !armed || body == nil || !p.matchesRootID(&c.rootDisplay.rootID) || !p.rootID.active || !c.rootDisplay.active || !z.MatchesCopyAllocationInvocation(&p.rootID.initialCopy) {
		t.Fatal("root display did not retain accepted root-ID copy", body, armed, sink.String())
	}
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	if r.flags>>3 != 0 {
		t.Fatal("failed root-ID published cache")
	}
	source := p.rootID.sourceCursor
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, c.CleanupRetained())
	if p.rootID.active || c.rootDisplay.active || p.rootID.sourceCursor != source {
		t.Fatal("display child acknowledgement changed source cursor")
	}
	serviceMust(t, c.DisplayRootInto(&algebraicRootTextSink{}, &root))
	l, e = c.location(&root)
	serviceMust(t, e)
	r, e = c.load(l)
	serviceMust(t, e)
	if r.flags>>3 != 1 {
		t.Fatal("independent first positive-root index", r.flags>>3)
	}
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialRootIDPreservesOlderCopy(t *testing.T) {
	c, p, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := p.config.Integer
	var old unumeral.IntegerCopyAllocationReceipt
	var destination unumeral.Int
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &old)
	if body == nil || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("older copy did not retain actual allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	prior, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	var receipt upolyRootIDInvocationReceipt
	_, body = p.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	if body == nil || !p.matchesRootID(&receipt) || p.rootID.initialCopy != (unumeral.IntegerCopyAllocationReceipt{}) || p.rootID.active {
		t.Fatal("refused initial SET adopted older integer copy", body)
	}
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if prior != z.CopyAllocation || !z.MatchesCopyAllocationInvocation(&old) || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("root-ID cleanup consumed older copy")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&old))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.Del(&root))
}
func TestUPolynomialRootIDPreservesOlderHorner(t *testing.T) {
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
	serviceMust(t, q.sturm(v))
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignBinary(v, r.interval.BorrowLower())
	if body == nil || !p.binaryEvaluation.active {
		t.Fatal("older public Horner did not retain allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	prior := p.binaryEvaluation
	_, body = q.evaluate(r.interval.BorrowLower())
	body = q.finish(body)
	if body == nil || q.evaluation.owner != nil || q.childStop != nil || q.active || !reflect.DeepEqual(prior, p.binaryEvaluation) {
		t.Fatal("root-ID refused evaluation adopted older Horner", body)
	}
	serviceMust(t, p.cleanupRootIDWithReceipt(&receipt))
	if !reflect.DeepEqual(prior, p.binaryEvaluation) {
		t.Fatal("root-ID cleanup consumed older Horner")
	}
	serviceMust(t, p.CleanupBinaryEvaluation())
	serviceMust(t, c.Del(&root))
}
