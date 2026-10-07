// Source: src/math/polynomial/polynomial.cpp:3432-3534,5356-5480;
// src/util/obj_ref.h; src/util/memory_manager.cpp.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This actual deletion callback selects the guarded signed-host TLS domain.
// It does not model a defined C++ allocation failure or a mathematical defect.
func TestPolynomialResultantICCPStoppedChildPrecedesOlderLocals(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: runtime, Profile: profile, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: runtime, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	for j := 0; j < 2; j++ {
		_, e := mm.MkVar()
		serviceMust(t, e)
	}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	p := polynomialICCPUnownedTerms(t, c, []int32{1, 1, 1, 2, 1, 3}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
	b := polynomialICCPUnownedTerms(t, c, []int32{1, 5, 1, 6}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, pr.Assign(p))
	br, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, br.Assign(b))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var oldICCP polynomialICCPInvocationReceipt
	var integerContent util.Int
	var cc, prim polynomialReference
	serviceMust(t, m.iccpPartsWithReceipt(polynomialICCPUnownedTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 1}}}), 0, &integerContent, &cc, &prim, &oldICCP))
	serviceMust(t, cc.close(c))
	serviceMust(t, prim.close(c))
	serviceMust(t, c.integer.Del(&integerContent))
	var lane util.Block
	calls := 0
	h, e := NewPolynomialDeletionHandler(budget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, p Polynomial) error {
		if len(m.iccpFrames) == 0 {
			return nil
		}
		s := m.iccpFrames[0]
		if !s.active || !s.ciConstructed || s.ci.value != p || calls != 0 {
			return nil
		}
		calls++
		contextTLSNearMinimum(t, runtime)
		var e error
		lane, e = runtime.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000) - runtime.State().LocalSize
		request := uint64(16) + uint64(delta) - 8
		if e = runtime.Reallocate(&lane, request); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("real oversized realloc", e)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	e = m.Resultant(p, b, 0, out)
	if !errors.Is(e, arena.ErrBounds) || calls != 1 || len(m.iccpFrames) == 0 || !m.iccpFrames[0].active || m.iccpFrames[0].cleanupCause == nil {
		t.Fatal("actual reached ICCP child source stop", e, calls)
	}
	child := m.iccpFrames[0]
	parent := m.gcdFrames[0]
	oldA, oldB := parent.refs[0].root.value, parent.refs[1].root.value
	if oldA != p || oldB != b || parent.depth == 0 {
		t.Fatal("older source result locals not retained", oldA, oldB, parent.depth)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{pr.root.value, br.root.value, out.root.value}, nil, pr.header+br.header+out.header+h.header, []util.Block{lane})
	receipt := parent.iccpInvocation
	if !m.matchesICCPInvocation(&receipt) || receipt.frame != oldICCP.frame || receipt.invocation == oldICCP.invocation {
		t.Fatal("exact retained ICCP invocation", receipt, oldICCP)
	}
	beforeUsed, beforeState := budget.Used(), runtime.State()
	serviceMust(t, m.cleanupICCPInvocation(&oldICCP, false))
	serviceMust(t, m.cleanupICCPInvocation(&oldICCP, true))
	foreign := receipt
	foreign.manager = &MultivariatePolynomialManager{}
	serviceMust(t, m.cleanupICCPInvocation(&foreign, true))
	if !child.active || runtime.State() != beforeState || budget.Used() != beforeUsed || calls != 1 {
		t.Fatal("stale/foreign receipt consumed replacement child")
	}
	rejected := receipt
	if e := m.iccpPartsWithReceipt(p, 0, &integerContent, &cc, &prim, &rejected); !errors.Is(e, ErrPolynomialPending) || rejected != receipt {
		t.Fatal("guard rejection changed callee receipt", e)
	}
	if m.iccpInvocationSourceCause(&receipt) != child.cleanupCause || m.iccpInvocationSourceCause(&oldICCP) != nil {
		t.Fatal("source cause identity mismatch")
	}
	cause := child.cleanupCause
	state := runtime.State()
	if retry := m.iccpSourceEnd(child); retry != cause || runtime.State() != state {
		t.Fatal("source retry replayed stopped child", retry, cause)
	}
	// The exact owning continuation must acknowledge the child before releasing
	// older A/B locals. Leaving the child active after erasing those is the RED.
	contextTLSFinish(t, runtime, m.CleanupGCD)
	if child.active {
		t.Errorf("parent owning cleanup erased older locals while matched ICCP child stayed active: A=%v B=%v", parent.refs[0].root.value, parent.refs[1].root.value)
	}
	// Baseline teardown explicitly owns the still-retained child; this does not
	// count as proof that CleanupGCD itself completed the composed protocol.
	contextTLSFinish(t, runtime, m.CleanupContentPrimitiveParts)
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	for _, ref := range []*PolynomialRef{pr, br, out} {
		polynomialTLSCloseReference(t, m, ref, func() {
			polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{pr.root.value, br.root.value, out.root.value}, nil, pr.header+br.header+out.header, []util.Block{lane})
		})
	}
	contextTLSFinish(t, runtime, func() error {
		if lane == (util.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
	polynomialTLSCloseFull(t, m, runtime, mm)
	if e := mm.DecManagerRef(); e != nil && !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	contextTLSFinish(t, runtime, mm.CloseChecked)
	contextTLSFinish(t, runtime, q.CloseChecked)
	contextTLSFinish(t, runtime, runtime.Close)
	if budget.Used() != 0 {
		t.Fatal("actual remaining charged roots", budget.Used())
	}
}

// This fixture owns inputs through the explicit public references only.
func polynomialICCPUnownedTerms(t *testing.T, m *MultivariatePolynomialCore, as []int32, pws [][]MonomialPower) Polynomial {
	t.Helper()
	coeffs := make([]util.Int, len(as))
	terms := make([]Monomial, len(as))
	for j := range as {
		coeffs[j] = util.NewInt(as[j])
		var e error
		terms[j], e = m.monomials.MkMonomial(pws[j])
		serviceMust(t, e)
	}
	p, e := m.MkTerms(coeffs, terms)
	serviceMust(t, e)
	return p
}

// Capture only the explicit caller's children before its actual destruction.
// A replacement ID-vector block cannot hide consumption of one of those tokens.
func polynomialTLSCloseReference(t *testing.T, m *MultivariatePolynomialManager, ref *PolynomialRef, audit func()) {
	t.Helper()
	c := m.core
	var cell *polynomialLocation
	var children []*monomialLocation
	if ref.root.value != (Polynomial{}) {
		var e error
		cell, e = c.polynomialLocation(ref.root.value)
		serviceMust(t, e)
		n, e := c.readWord(ref.root.value, 8)
		serviceMust(t, e)
		for j := uint32(0); j < n; j++ {
			term, e := c.monomialCore(ref.root.value, j)
			serviceMust(t, e)
			l, e := c.monomials.location(term)
			serviceMust(t, e)
			children = append(children, l)
		}
	}
	for attempts := 0; ; attempts++ {
		if attempts > 64 {
			t.Fatal("bounded caller continuation")
		}
		var raws []util.ObjectRef
		for _, l := range children {
			raws = append(raws, l.raw)
		}
		cellReleased := cell != nil && cell.released
		before := c.memory.Stats().LiveBlocks
		e := ref.Close()
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual caller close", e)
		}
		consumed := false
		if cell != nil && !cellReleased && cell.released {
			if _, e := c.monomials.allocator.ReadUint32(cell.raw, 0); e == nil {
				t.Fatal("consumed caller allocation readable")
			}
			consumed = true
		}
		for j, l := range children {
			if raws[j] != (util.ObjectRef{}) && l.raw == (util.ObjectRef{}) {
				if _, e := c.monomials.allocator.ReadUint32(raws[j], 0); e == nil {
					t.Fatal("consumed child allocation readable")
				}
				consumed = true
			}
		}
		audit()
		if !consumed && c.memory.Stats().LiveBlocks >= before {
			t.Fatal("no exact caller allocation progress", e)
		}
	}
}
