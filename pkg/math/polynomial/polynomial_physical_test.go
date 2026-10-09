// Source: src/math/polynomial/polynomial.cpp:1885-2338,2481-2633.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Independent named field census for this selected owner and its ordinary
// unsynchronized borrowed rational cohort. This does not infer caller roots
// from the locator directory, nor cover unrelated rational arithmetic scopes.
func polynomialPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, public ...*util.Int) {
	t.Helper()
	polynomialPhysicalAuditWithHandlers(t, m, nil, public...)
}
func polynomialPhysicalAuditWithHandlers(t *testing.T, m *MultivariatePolynomialCore, handlers []*PolynomialDeletionHandler, public ...*util.Int) {
	t.Helper()
	z := m.integer
	refs := map[util.NumericCellRef]bool{}
	add := func(a util.Int) {
		if (a).Cell == 0 || z.IsFailureRef((a).Cell) {
			return
		}
		if !z.IsLiveCellToken(uint64((a).Cell)) {
			t.Fatal("invalid named numeral", a)
		}
		refs[(a).Cell] = true
	}
	for _, r := range z.Tmp {
		add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(r), uint32(uint64(r) >> 32)}))
	}
	for _, r := range z.Arg {
		add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(r), uint32(uint64(r) >> 32)}))
	}
	add(z.IntMin)
	add(z.Two64)
	public = append(public, &m.substitution.scope.newA, &m.substitution.scope.tmp)
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		public = append(public, &b.cohortMovingCoefficient, &b.appendScope.movingCoefficient, &b.arithmeticScope.transport)
	}
	for _, a := range public {
		if a != nil {
			add(*a)
		}
	}
	q := m.config.Rationals
	if q != nil {
		for _, a := range []util.Int{q.NTmp, q.AddTmp1, q.AddTmp2, q.AddMulTmp.Num, q.AddMulTmp.Den, q.LtTmp1.Num, q.LtTmp1.Den, q.LtTmp2.Num, q.LtTmp2.Den} {
			add(a)
		}
	}
	for _, a := range []util.Int{(*m.modular.BorrowModulus()), m.modular.Lower, m.modular.Upper, m.modular.InvTmp1, m.modular.InvTmp2, m.modular.InvTmp3, m.modular.DivTmp, m.zeroNumeral, m.minusOne, m.negativeCoefficient, m.constantCopy, m.linearConstant, m.constantScope.numeral, (*m.constantScope.rational.BorrowNumerator()), (*m.constantScope.rational.BorrowDenominator()), m.exactDivision.numeral, m.pseudoDivision.numeral} {
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
	privatePolys := polynomialSubstitutionNamedPolynomials(m)
	privatePolys = append(privatePolys, m.translation.freshResult)
	for i := range m.pseudoDivision.refs {
		privatePolys = append(privatePolys, m.pseudoDivision.refs[i].value, m.pseudoDivision.refs[i].incoming)
	}
	if m.powerLocal.active {
		privatePolys = append(privatePolys, m.powerLocal.result.value, m.powerLocal.result.incoming)
	}
	for _, p := range privatePolys {
		if p == (Polynomial{}) {
			continue
		}
		l, e := m.polynomialLocation(p)
		serviceMust(t, e)
		if !l.released {
			cell(p, l.initializedCoefficients)
		}
	}
	if m.pending.value != (Polynomial{}) {
		l, e := m.polynomialLocation(m.pending.value)
		serviceMust(t, e)
		cell(m.pending.value, l.initializedCoefficients)
	}
	vectors := []*polynomialNumerals{&m.rationalTerms, &m.linearTerms}
	buffers := [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2}
	for _, b := range buffers {
		add(b.gcd)
		vectors = append(vectors, &b.as)
	}
	for _, v := range vectors {
		for _, a := range v.values {
			add(a)
		}
		for _, a := range v.retired {
			add(a)
		}
	}
	for slot := uint64(1); slot <= uint64(z.Cells.Stats().Slots); slot++ {
		c, e := z.Cells.Read(arena.Slot(slot))
		if e == nil && (c.Unpublished || c.CleanupPending) {
			add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(util.MakeNumericRef(arena.Slot(slot), z.Cells.State(arena.Slot(slot)).Generation)), uint32(uint64(util.MakeNumericRef(arena.Slot(slot), z.Cells.State(arena.Slot(slot)).Generation)) >> 32)}))
		}
	}
	if uint32(len(refs)) != z.Cells.Stats().Live {
		t.Fatal("unnamed actual numeric descriptor", len(refs), z.Cells.Stats().Live)
	}
	for ref := range refs {
		c := z.Cell(ref)
		if c.Payload != (arena.VectorRef{}) {
			t.Fatal("fake Words cell")
		}
		if !c.Released {
			o, e := z.CellObject(ref)
			serviceMust(t, e)
			_, e = z.SourceAllocator().ObjectExtent(o)
			serviceMust(t, e)
		}
	}
	if z.PendingAllocationObject() != (util.ObjectRef{}) {
		_, e := z.SourceAllocator().ObjectExtent(z.PendingAllocationObject())
		serviceMust(t, e)
	}
	raw := map[util.Block]bool{}
	put := func(b util.Block) {
		if b == (util.Block{}) {
			return
		}
		if raw[b] {
			t.Fatal("duplicate named physical root")
		}
		raw[b] = true
	}
	for _, b := range polynomialSubstitutionNamedRaw(m) {
		put(b)
	}
	mm := m.monomials

	za, ma := z.SourceAllocator().Stats(), mm.allocator.Stats()
	if uint32(len(raw))+za.Chunks+za.DirectObjects+za.PendingBlocks+ma.Chunks+ma.DirectObjects+ma.PendingBlocks != m.memory.Stats().LiveBlocks {
		t.Fatal("unnamed Runtime root", len(raw), za, ma, m.memory.Stats())
	}
	bytes := uint64(cap(m.substitution.orphans))*16 + uint64(m.ids.FreeCapacity()+mm.ids.FreeCapacity())*4 + integerActualOwnedBytes(z) + m.header + mm.header + uint64(cap(mm.locations))*uint64(reflect.TypeOf(monomialLocation{}).Size()) + ma.TotalBytes + uint64(cap(m.locations))*uint64(reflect.TypeOf(polynomialLocation{}).Size())
	if !m.modular.Closed() {
		bytes += m.modular.Header
	}
	if q != nil {
		bytes += q.HeaderBytes
	}
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		bytes += uint64(cap(b.coefficientOwners)+cap(b.monomialOwners)+cap(b.coefficientTransferred)+cap(b.monomialTransferred)) * 8
	}
	for _, h := range handlers {
		if h == nil || h.self != h || h.budget != z.Budget() {
			t.Fatal("invalid named handler")
		}
		bytes += h.header
	}
	if bytes != z.Budget().Used() {
		t.Fatal("actual charged headers and backings", bytes, z.Budget().Used())
	}
}
