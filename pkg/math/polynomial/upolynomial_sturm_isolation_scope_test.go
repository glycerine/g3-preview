package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

func TestUPolynomialSturmIsolationStoppedStackProtectsOlderLocals(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	b := c.config.Binary
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	p := upolySourceNew(t, m, []int32{-6, 11, -6, 1})
	v := mustUPolyView(t, m, p)
	z := m.config.Integer
	reached := false
	var ref unumeral.NumericCellRef
	var requested uint64
	m.config.Checkpoint = func() bool {
		q := &m.sturmIsolation
		if reached || !q.executing || !q.live[4] || len(q.stack.values) == 0 {
			return true
		}
		ref = q.stack.values[0].upper.Num.Cell
		if !unumeral.ValidNumericRef(z.Cells, ref) {
			return true
		}
		reached = true
		used, stackSize := z.Budget().Used(), len(q.stack.values)
		for _, e := range []error{q.setPrepared(UPolynomialView{}), m.pushSSFrameWithIsolation(UPolynomialView{}, b, &q.stack, roots, lo, hi, 3, 1, nil, nil, 2463, q)} {
			if !errors.Is(e, ErrUPolynomialOwner) {
				t.Fatal("checkpoint acquired paused native mutation", e)
			}
		}
		if used != z.Budget().Used() || stackSize != len(q.stack.values) {
			t.Fatal("refused scope entry changed storage")
		}
		for _, e := range []error{z.CloseChecked(), m.modular.Close(), b.CloseChecked()} {
			if !errors.Is(e, unumeral.ErrNumericPending) {
				t.Fatal("checkpoint destroyed borrowed numerical owner", e)
			}
		}
		if e := z.Naturals().CloseChecked(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("checkpoint destroyed borrowed natural owner", e)
		}
		cell := z.Cell(ref)
		requested = cell.Backing.Requested
		cell.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(ref, cell)
		for _, e := range []error{m.Close(), m.SourceFinalize(), m.CleanupSturmIsolation(), m.DelPolynomial(&p)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("checkpoint acquired executing isolation owner", e)
			}
		}
		return false
	}
	m.config.Cooperation = func() error { return nil }
	body := m.SturmIsolateRootsWithBounds(v, 128, 128, b, roots, lo, hi)
	q := &m.sturmIsolation
	if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || !q.active || q.executing || q.sourceStage != 0 || q.sourceStack != 1 || q.sourceCoefficient != 0 || q.live != ([5]bool{true, true, true, true, true}) || q.sourceStop == nil || z.Cell(ref).Released {
		t.Fatal("stack source stop lost old locals", body, q.sourceStage, q.sourceStack, q.live)
	}
	for _, a := range []*unumeral.Int{&q.lower.Num, &q.upper.Num} {
		if !unumeral.ValidNumericRef(z.Cells, a.Cell) || z.Cell(a.Cell).Released {
			t.Fatal("older bound destroyed after stopped stack")
		}
	}
	before, used, stats := *q, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("manager crossed stopped stack", e)
	}
	if e := q.finishSource(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("source replay changed stopped history")
	}
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, m.CleanupSturmIsolation())
	if q.active || q.pending() || q.sourceStage != 0 || q.sourceStack != 1 || q.sourceCoefficient != 0 || q.ownStage != 5 {
		t.Fatal("owning cleanup rewrote source history")
	}
	previousSequenceInvocation := q.sequence.sturm.invocation
	m.config.Checkpoint = func() bool { return true }
	serviceMust(t, m.SturmIsolateRootsWithBounds(v, 3, 3, b, roots, lo, hi))
	if q.sequence.sturm.invocation <= previousSequenceInvocation {
		t.Fatal("internal sequence reused retired numeric invocation")
	}

	counts := upolyIsolationCounts{}
	upolyIsolationMatch(t, c.config.Rationals, b, roots, lo, hi, []upolyIsolationRational{{1, 1}, {2, 1}, {3, 1}}, &counts)
	for _, output := range []*BinaryRationalVector{roots, lo, hi} {
		serviceMust(t, output.Reset())
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{p}, nil, nil)
}

func TestUPolynomialSturmIsolationRetainsConstructorCopy(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	b := c.config.Binary
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := m.SturmIsolateRootsWithBounds(v, 3, 3, b, roots, lo, hi)
	q := &m.sturmIsolation
	if body == nil || !q.active || !q.sequence.sturm.active || !m.config.Integer.MatchesCopyAllocationInvocation(&q.sequence.sturm.copy) {
		t.Fatal("actual internal constructor SET receipt lost", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	source, coefficient := q.sourceStage, q.sourceCoefficient
	serviceMust(t, m.CleanupSturmIsolation())
	if q.active || q.pending() || q.sourceStage != source || q.sourceCoefficient != coefficient || m.sturmPending != nil {
		t.Fatal("constructor acknowledgement changed source history")
	}
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSturmIsolationReachedReservations(t *testing.T) {
	success, failed, constructorDebt := false, false, false
	for after := int64(0); after < 1024 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			b := c.config.Binary
			roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			p := upolySourceNew(t, m, []int32{-6, 11, -6, 1})
			z := m.config.Integer
			var input [4]unumeral.Int
			for i, text := range []string{"-110680464442257309696", "202914184810805067776", "-110680464442257309696", "18446744073709551616"} {
				serviceMust(t, z.SetString(&input[i], text))
			}
			serviceMust(t, m.SetInts(p, input[:]))
			for i := range input {
				serviceMust(t, z.Del(&input[i]))
			}
			z.Budget().FailAfter(after)
			body := m.SturmIsolateRootsWithBounds(mustUPolyView(t, m, p), 3, 3, b, roots, lo, hi)
			z.Budget().FailAfter(-1)
			q := &m.sturmIsolation
			if q.sequence.sturm.active {
				constructorDebt = true
			}
			if body == nil {
				success = true
				counts := upolyIsolationCounts{}
				upolyIsolationMatch(t, c.config.Rationals, b, roots, lo, hi, []upolyIsolationRational{{1, 1}, {2, 1}, {3, 1}}, &counts)
			} else {
				failed = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected isolation refusal", body)
				}
			}
			stage, stack, coefficient := q.sourceStage, q.sourceStack, q.sourceCoefficient
			serviceMust(t, m.CleanupSturmIsolation())
			if q.active || q.pending() || stage != q.sourceStage || stack != q.sourceStack || coefficient != q.sourceCoefficient || m.sturmPending != nil {
				t.Fatal("isolation cleanup changed source history")
			}
			for _, v := range []*BinaryRationalVector{roots, lo, hi} {
				serviceMust(t, v.Reset())
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{p}, nil, nil)
		})
	}
	if !success || !failed || !constructorDebt {
		t.Fatal("isolation reservation campaign incomplete", success, failed, constructorDebt)
	}
}

func TestUPolynomialSturmIsolationOuterRetainsPreparedVector(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	b := c.config.Binary
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	writer := &rootIDRemainderDisplayStopWriter{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("rem_bug")
	serviceMust(t, m.config.Integer.BindTrace(tags))
	body := m.SturmIsolateRoots(v, b, roots, lo, hi)
	serviceMust(t, m.config.Integer.BindTrace(nil))
	q, z := &m.sturmIsolation, m.config.Integer
	if tags.ShouldTrace("rem_bug") {
		if !writer.reached || !errors.Is(body, arena.ErrBounds) || !q.active || q.childStop == nil || !q.preparedLive || q.preparedSource != 0 || q.sourceStage != 0 {
			t.Fatal("inner source stop destroyed older preparation", body, q.sourceStage)
		}
		for _, n := range m.values(&q.prepared)[:q.prepared.size] {
			if n.Cell != 0 && z.Cell(n.Cell).Released {
				t.Fatal("prepared coefficient destroyed")
			}
		}
		cell := z.Cell(writer.ref)
		cell.Backing.Requested = writer.requested
		z.Store(writer.ref, cell)
		serviceMust(t, m.CleanupSturmIsolation())
		if q.active || q.preparedLive || q.preparedSource != 0 || q.preparedOwn != 3 || q.sourceStage != 0 {
			t.Fatal("outer owning cleanup rewrote source history")
		}
	} else {
		serviceMust(t, body)
	}
	for _, v := range []*BinaryRationalVector{roots, lo, hi} {
		serviceMust(t, v.Reset())
	}
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSturmIsolationOuterEarlyCopyOwnership(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	b := c.config.Binary
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := m.SturmIsolateRoots(v, b, roots, lo, hi)
	q := &m.sturmIsolation
	if body == nil || !q.active || q.sequence.self != nil || !m.config.Integer.MatchesCopyAllocationInvocation(&q.copy) {
		t.Fatal("outer early SET lost exact owner", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	source := q.sourceStage
	serviceMust(t, m.CleanupSturmIsolation())
	if q.active || q.pending() || q.sourceStage != source {
		t.Fatal("early-copy cleanup changed source history")
	}
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}
