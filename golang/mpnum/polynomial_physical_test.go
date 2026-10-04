// Source: src/math/polynomial/polynomial.cpp:1885-2338,2481-2633. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

// Independent named field census for this selected owner and its ordinary
// unsynchronized borrowed rational cohort. This does not infer caller roots
// from the locator directory, nor cover unrelated rational arithmetic scopes.
func polynomialPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, public ...*Int) {
	t.Helper()
	polynomialPhysicalAuditWithHandlers(t, m, nil, public...)
}
func polynomialPhysicalAuditWithHandlers(t *testing.T, m *MultivariatePolynomialCore, handlers []*PolynomialDeletionHandler, public ...*Int) {
	t.Helper()
	z := m.integer
	refs := map[numericCellRef]bool{}
	add := func(a Int) {
		if a.cell == 0 || z.isFailureRef(a.cell) {
			return
		}
		if !validNumericRef(z.cells, a.cell) {
			t.Fatal("invalid named numeral", a)
		}
		refs[a.cell] = true
	}
	for _, r := range z.tmp {
		add(Int{cell: r})
	}
	for _, r := range z.arg {
		add(Int{cell: r})
	}
	add(z.intMin)
	add(z.two64)
	for _, a := range public {
		if a != nil {
			add(*a)
		}
	}
	q := m.config.Rationals
	if q != nil {
		for _, a := range []Int{q.nTmp, q.addTmp1, q.addTmp2, q.addMulTmp.num, q.addMulTmp.den, q.ltTmp1.num, q.ltTmp1.den, q.ltTmp2.num, q.ltTmp2.den} {
			add(a)
		}
	}
	for _, a := range []Int{m.modular.p, m.modular.lower, m.modular.upper, m.modular.invTmp1, m.modular.invTmp2, m.modular.invTmp3, m.modular.divTmp, m.zeroNumeral, m.minusOne, m.negativeCoefficient, m.constantCopy, m.linearConstant, m.constantScope.numeral, m.constantScope.rational.num, m.constantScope.rational.den, m.exactDivision.numeral} {
		add(a)
	}
	cell := func(p Polynomial, n uint32) {
		for i := uint32(0); i < n; i++ {
			a, e := m.coefficientCore(p, i)
			serviceMust(t, e)
			add(a)
		}
	}
	for _, p := range []Polynomial{m.zero, m.unit, m.constantScope.result, m.overloadResult, m.negativeResult, m.exactDivision.returned} {
		if p != (Polynomial{}) && !(p == m.zero && m.sourceStage > 0) && !(p == m.unit && m.sourceStage > 1) {
			n, e := m.readWord(p, 8)
			serviceMust(t, e)
			cell(p, n)
		}
	}
	if m.pending.value != (Polynomial{}) {
		l, e := m.polynomialLocation(m.pending.value)
		serviceMust(t, e)
		cell(m.pending.value, l.initializedCoefficients)
	}
	vectors := []*polynomialVector{&m.rationalTerms, &m.linearTerms, &m.linearMonomials, &m.polynomials}
	buffers := [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2}
	for _, b := range buffers {
		add(b.gcd)
		vectors = append(vectors, &b.as, &b.ms)
	}
	for _, v := range vectors {
		if v.width != 12 {
			continue
		}
		if v.raw != (memory.Block{}) {
			for i := uint32(0); i < v.initialized; i++ {
				a, e := v.coefficient(m, i)
				serviceMust(t, e)
				add(a)
			}
		}
		if v.old != (memory.Block{}) {
			old := polynomialVector{raw: v.old, width: 12}
			for i := v.moved; i < v.oldInitialized; i++ {
				a, e := old.coefficient(m, i)
				serviceMust(t, e)
				add(a)
			}
		}
		for i := uint32(0); i < v.retiredActive; i++ {
			add(v.retired[i])
		}
	}
	for slot := uint64(1); slot <= uint64(z.cells.Stats().Slots); slot++ {
		c, e := z.cells.Read(arena.Slot(slot))
		if e == nil && (c.unpublished || c.cleanupPending) {
			add(Int{cell: makeNumericRef(arena.Slot(slot), z.cells.state(arena.Slot(slot)).generation)})
		}
	}
	if uint32(len(refs)) != z.cells.Stats().Live {
		t.Fatal("unnamed actual numeric descriptor", len(refs), z.cells.Stats().Live)
	}
	for ref := range refs {
		c := z.cell(ref)
		if c.payload != (arena.VectorRef{}) {
			t.Fatal("fake Words cell")
		}
		if !c.released {
			o, e := z.cellObject(ref)
			serviceMust(t, e)
			_, e = z.allocator.ObjectExtent(o)
			serviceMust(t, e)
		}
	}
	if z.pendingCell.object != (memory.ObjectRef{}) {
		_, e := z.allocator.ObjectExtent(z.pendingCell.object)
		serviceMust(t, e)
	}
	raw := map[memory.Block]bool{}
	put := func(b memory.Block) {
		if b == (memory.Block{}) {
			return
		}
		if raw[b] {
			t.Fatal("duplicate named physical root")
		}
		raw[b] = true
	}
	mm := m.monomials
	for _, r := range []memory.Block{mm.table.raw, mm.table.pending, mm.renameTable.raw, mm.renameTable.pending, mm.renameFound, mm.powers.raw, mm.powers.pending} {
		put(r)
	}
	for _, tmp := range mm.temps {
		put(tmp.raw)
		put(tmp.pending)
	}
	for _, r := range mm.ids.RawRoots() {
		put(r)
	}
	for _, r := range m.ids.RawRoots() {
		put(r)
	}
	for _, v := range vectors {
		put(v.raw)
		put(v.pending)
		put(v.old)
	}
	for _, b := range buffers {
		put(b.positions.raw)
		put(b.positions.pending)
	}
	for _, s := range z.natural.division {
		put(s.raw)
		put(s.pending)
	}
	if z.natural.verificationActive {
		put(z.natural.verification.raw)
		put(z.natural.verification.pending)
	}
	if z.natural.formatting.active || z.natural.formatting.pending {
		for _, s := range z.natural.formatting.locals {
			put(s.raw)
			put(s.pending)
		}
	}
	put(z.natural.charFormatting.buffer.raw)
	put(z.natural.charFormatting.buffer.pending)
	za, ma := z.allocator.Stats(), mm.allocator.Stats()
	if uint32(len(raw))+za.Chunks+za.DirectObjects+za.PendingBlocks+ma.Chunks+ma.DirectObjects+ma.PendingBlocks != m.memory.Stats().LiveBlocks {
		t.Fatal("unnamed Runtime root", len(raw), za, ma, m.memory.Stats())
	}
	bytes := uint64(cap(m.linearGenerated)) + integerActualOwnedBytes(z) + m.header + mm.header + uint64(cap(mm.locations))*uint64(reflect.TypeOf(monomialLocation{}).Size()) + ma.TotalBytes + uint64(cap(m.locations))*uint64(reflect.TypeOf(polynomialLocation{}).Size())
	if !m.modular.closed {
		bytes += m.modular.header
	}
	if q != nil {
		bytes += q.headerBytes
	}
	for _, v := range vectors {
		bytes += uint64(cap(v.retired)) * uint64(reflect.TypeOf(Int{}).Size())
	}
	for _, h := range handlers {
		if h == nil || h.self != h || h.budget != z.budget {
			t.Fatal("invalid named handler")
		}
		bytes += h.header
	}
	if bytes != z.budget.Used() {
		t.Fatal("actual charged headers and backings", bytes, z.budget.Used())
	}
}
