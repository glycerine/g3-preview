package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestUPolynomialSequenceSturmRemainderScalarStopProtectsVectors(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	z := m.config.Integer
	reached := false
	var ref unumeral.NumericCellRef
	var requested uint64
	m.config.Checkpoint = func() bool {
		q := &s.sturm
		if reached || !q.scalar.constructed {
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
	m.config.Cooperation = func() error { return nil }
	body := m.SturmSequence(v, s)
	q := &s.sturm
	if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || q.scalar.sourceStop == nil || q.childStop == nil || !q.scalar.constructed || q.scalar.sourceCursor != 0 || q.sourceCursor != ([2]uint32{}) || !q.live[0] || !q.live[1] || z.Cell(ref).Released {
		t.Fatal("scalar stop lost body/older vector order", body, q.sourceCursor)
	}
	for _, state := range []*upolyVectorState{&q.remainder, &s.state.coefficients} {
		for _, value := range m.values(state)[:state.size] {
			if value.Cell != 0 && z.Cell(value.Cell).Released {
				t.Fatal("older vector coefficient destroyed")
			}
		}
	}
	before, used, stats := q.scalar, z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source manager lost stopped Sturm child", e)
	}
	if e := q.finishRemainderScalar(nil); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, q.scalar) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("stopped scalar source finalizer replayed")
	}
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, s.CleanupSturm())
	if m.sturmPending != nil {
		t.Fatal("owning cleanup retained acknowledged invocation")
	}
	if q.active || q.scalar.constructed || q.scalar.sourceCursor != 0 || q.scalar.ownCursor != 1 || q.sourceCursor != ([2]uint32{}) {
		t.Fatal("owning continuation changed source cursors")
	}
	m.config.Checkpoint = func() bool { return true }
	serviceMust(t, m.SturmSequence(v, s))
	minus, e := m.SignVariationsMinusInfinity(s)
	serviceMust(t, e)
	plus, e := m.SignVariationsPlusInfinity(s)
	serviceMust(t, e)
	if minus-plus != 2 {
		t.Fatal("independent two real roots", minus, plus)
	}
	serviceMust(t, s.Close())
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSequenceSturmNumericContextRejectsCheckpointAndRetirement(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v, e := m.wholeView(p)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	z := m.config.Integer
	var context integerAllocationInvocationContext
	probe := unumeral.NewInt(0)
	probed := false
	m.config.Checkpoint = func() bool {
		if probed {
			return true
		}
		probed = true
		context = s.sturm.context()
		used := z.Budget().Used()
		var untouched upolyVectorState
		d := uint32(77)
		for _, e := range []error{m.setViewWithSourceContext(&untouched, v, context), m.derivativeStateWithSourceContext(v, &untouched, context), m.mulWithSourceContext(v, v, &untouched, nil, context), m.remStateWithSourceContext(v, v, &d, &untouched, context), m.signedRemStateWithSourceContext(v, v, &untouched, context)} {
			if !errors.Is(e, ErrUPolynomialOwner) {
				t.Fatal("checkpoint acquired numeric context before native storage", e)
			}
		}
		if untouched != (upolyVectorState{}) || d != 77 || context.sequenceSturmFrame(z) != nil || s.sturm.NumericRoots(z, nil) != nil {
			t.Fatal("paused numeric entry mutated source state")
		}
		if e := z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7); e == nil || probe.Val != 0 || probe.Cell != 0 || used != z.Budget().Used() {
			t.Fatal("checkpoint borrowed executing numerical context", e)
		}
		return true
	}
	m.config.Cooperation = func() error { return nil }
	serviceMust(t, m.SturmSequence(v, s))
	if !probed || context.sequenceSturmFrame(z) != nil {
		t.Fatal("retired invocation retained numeric authority")
	}
	if e := z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 9); e == nil || probe.Val != 0 {
		t.Fatal("retired context accepted numeric write", e)
	}
	serviceMust(t, s.Close())
}

func TestUPolynomialSequenceSturmScalarCopyExactAcknowledgement(t *testing.T) {
	var m *UPolynomialManager
	var s *UPolynomialSequence
	armed, probed := false, false
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		probed = true
		z, q := m.config.Integer, &s.sturm
		context := q.context()
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		probe := unumeral.NewInt(0)
		if context.sequenceSturmFrame(z) != nil || q.NumericRoots(z, nil) != nil {
			t.Error("failure transport acquired paused Sturm numeric context")
		}
		for _, ctx := range []unumeral.NumericAllocationContext{context.utility(z), unumeral.NumericAllocationContext{}.WithCaller(&q.hooks)} {
			if e := z.WithAllocationContext(ctx).SetUint32(&probe, 7); e == nil || probe.Val != 0 || probe.Cell != 0 {
				t.Error("failure transport wrote through Sturm tuple/policy", e, probe)
			}
		}
		var untouched upolyVectorState
		if e := m.setViewWithSourceContext(&untouched, q.owner.manager.stateView(&q.remainder), context); !errors.Is(e, ErrUPolynomialOwner) || untouched != (upolyVectorState{}) {
			t.Error("failure transport reserved a native vector", e)
		}
		if used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("refused failure callback changed owner resources")
		}
		return f
	}
	c, owner, runtime := algebraicNegTLSFixture(t, transport)
	m = owner
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e = m.NewSequence()
	serviceMust(t, e)
	var pad umemory.Block
	m.config.Checkpoint = func() bool {
		if !armed && s.sturm.scalar.constructed {
			pad = algebraicCellSourceTLSBoundary(t, c)
			armed = true
		}
		return true
	}
	m.config.Cooperation = func() error { return nil }
	body := m.SturmSequence(v, s)
	q, z := &s.sturm, m.config.Integer
	if !armed || !probed || body == nil || !q.active || !z.MatchesCopyAllocationInvocation(&q.scalar.copy) || q.scalar.constructed || q.scalar.sourceCursor != 1 {
		t.Fatal("actual scalar SET refusal lost exact ownership", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	armed = false
	source, scalarSource := q.sourceCursor, q.scalar.sourceCursor
	if e := m.SourceFinalize(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("source manager crossed pending scalar copy", e)
	}
	serviceMust(t, s.CleanupSturm())
	if q.active || z.MatchesCopyAllocationInvocation(&q.scalar.copy) || source != q.sourceCursor || scalarSource != q.scalar.sourceCursor {
		t.Fatal("owning copy acknowledgement changed source history")
	}
	m.config.Checkpoint = func() bool { return true }
	serviceMust(t, m.SturmSequence(v, s))
	minus, e := m.SignVariationsMinusInfinity(s)
	serviceMust(t, e)
	plus, e := m.SignVariationsPlusInfinity(s)
	serviceMust(t, e)
	if minus-plus != 2 {
		t.Fatal("independent real root count", minus, plus)
	}
	serviceMust(t, s.Close())
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}
